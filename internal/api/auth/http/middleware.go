package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/KhachikAstoyan/capstone/internal/api/auth"
	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDKey      contextKey = "user_id"
	HandleKey      contextKey = "handle"
	PermissionsKey contextKey = "permissions"
)

func AuthMiddleware(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token := parts[1]
			claims, err := jwtManager.ValidateAccessToken(token)
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, HandleKey, claims.Handle)
			ctx = context.WithValue(ctx, PermissionsKey, claims.Permissions)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware validates a Bearer token when present and attaches user id and JWT
// permission claims to the request context. Requests without Authorization proceed unchanged.
// Malformed headers or invalid tokens receive 401.
func OptionalAuthMiddleware(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token := parts[1]
			claims, err := jwtManager.ValidateAccessToken(token)
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, HandleKey, claims.Handle)
			ctx = context.WithValue(ctx, PermissionsKey, claims.Permissions)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return userID, ok
}

func GetHandleFromContext(ctx context.Context) (string, bool) {
	handle, ok := ctx.Value(HandleKey).(string)
	return handle, ok
}

func GetPermissionsFromContext(ctx context.Context) ([]string, bool) {
	permissions, ok := ctx.Value(PermissionsKey).([]string)
	return permissions, ok
}
