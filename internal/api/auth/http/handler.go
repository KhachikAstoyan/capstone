package http

import (
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/auth/service"
)

type Handler struct {
	service       service.Service
	secureCookies bool
}

func NewHandler(svc service.Service, secureCookies bool) *Handler {
	return &Handler{
		service:       svc,
		secureCookies: secureCookies,
	}
}

func (h *Handler) setRefreshTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60,
	})
}

func (h *Handler) clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func getRefreshTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}
