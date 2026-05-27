package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	authhttp "github.com/KhachikAstoyan/capstone/internal/api/auth/http"
)

type slidingWindow struct {
	mu         sync.Mutex
	timestamps []time.Time
}

func (w *slidingWindow) allow(limit int, window time.Duration) bool {
	now := time.Now()
	cutoff := now.Add(-window)

	w.mu.Lock()
	defer w.mu.Unlock()

	// Prune expired entries
	valid := w.timestamps[:0]
	for _, t := range w.timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	w.timestamps = valid

	if len(w.timestamps) >= limit {
		return false
	}
	w.timestamps = append(w.timestamps, now)
	return true
}

var windows sync.Map // key: string → *slidingWindow

func init() {
	// Periodically remove idle entries to prevent unbounded map growth.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			windows.Range(func(k, v any) bool {
				sw := v.(*slidingWindow)
				sw.mu.Lock()
				idle := len(sw.timestamps) == 0
				sw.mu.Unlock()
				if idle {
					windows.Delete(k)
				}
				return true
			})
		}
	}()
}

// SubmissionRateLimit returns middleware that limits authenticated users to
// `limit` requests within `window`. Unauthenticated requests pass through
// (auth middleware is expected to run before this).
func SubmissionRateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := authhttp.GetUserIDFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			key := userID.String() + ":" + r.URL.Path

			val, _ := windows.LoadOrStore(key, &slidingWindow{})
			sw := val.(*slidingWindow)

			if !sw.allow(limit, window) {
				writeRateLimited(w, window)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPRateLimit returns middleware that limits requests by client IP address,
// independent of authentication. It is intended for pre-auth endpoints
// (login, register, refresh, email verification) that no per-user limiter
// can protect, since those requests have no authenticated identity.
func IPRateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := "ip:" + clientIP(r) + ":" + r.URL.Path

			val, _ := windows.LoadOrStore(key, &slidingWindow{})
			sw := val.(*slidingWindow)

			if !sw.allow(limit, window) {
				writeRateLimited(w, window)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// writeRateLimited emits the standard 429 response.
func writeRateLimited(w http.ResponseWriter, window time.Duration) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", window.String())
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "rate limit exceeded, please slow down",
	})
}

// clientIP extracts the originating client address. It honors the first
// entry of X-Forwarded-For when present (the platform runs behind a reverse
// proxy) and otherwise falls back to the transport-level remote address.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first, _, ok := strings.Cut(xff, ","); ok {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(xff)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
