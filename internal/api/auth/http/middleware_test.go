package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/api/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()
	handle := "testuser"
	permissions := []string{"test.permission", "another.permission"}

	token, err := jwtManager.GenerateAccessToken(userID, handle, permissions)
	require.NoError(t, err)

	middleware := AuthMiddleware(jwtManager)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// Verify context values
		ctxUserID, ok := GetUserIDFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, userID, ctxUserID)

		ctxHandle, ok := GetHandleFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, handle, ctxHandle)

		ctxPermissions, ok := GetPermissionsFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, permissions, ctxPermissions)

		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_MissingAuthHeader(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	middleware := AuthMiddleware(jwtManager)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing authorization header")
}

func TestAuthMiddleware_InvalidAuthHeaderFormat(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	middleware := AuthMiddleware(jwtManager)

	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "missing bearer prefix",
			header: "some-token",
		},
		{
			name:   "wrong prefix",
			header: "Basic some-token",
		},
		{
			name:   "only bearer",
			header: "Bearer",
		},
		{
			name:   "extra parts",
			header: "Bearer token extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()

			middleware(handler).ServeHTTP(w, req)

			assert.False(t, handlerCalled)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	middleware := AuthMiddleware(jwtManager)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired token")
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	// Create manager with very short expiration
	jwtManager := auth.NewJWTManager("test-secret", 1*time.Millisecond, 7*24*time.Hour)
	userID := uuid.New()
	handle := "testuser"
	permissions := []string{"test.permission"}

	token, err := jwtManager.GenerateAccessToken(userID, handle, permissions)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	middleware := AuthMiddleware(jwtManager)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	// Generate token with one secret
	jwtManager1 := auth.NewJWTManager("secret-1", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()
	handle := "testuser"
	permissions := []string{"test.permission"}

	token, err := jwtManager1.GenerateAccessToken(userID, handle, permissions)
	require.NoError(t, err)

	// Try to validate with different secret
	jwtManager2 := auth.NewJWTManager("secret-2", 15*time.Minute, 7*24*time.Hour)
	middleware := AuthMiddleware(jwtManager2)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetUserIDFromContext_Present(t *testing.T) {
	userID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, UserIDKey, userID)

	gotUserID, ok := GetUserIDFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, userID, gotUserID)
}

func TestGetUserIDFromContext_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := req.Context()

	_, ok := GetUserIDFromContext(ctx)
	assert.False(t, ok)
}

func TestGetUserIDFromContext_WrongType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, UserIDKey, "not-a-uuid")

	_, ok := GetUserIDFromContext(ctx)
	assert.False(t, ok)
}

func TestGetHandleFromContext_Present(t *testing.T) {
	handle := "testuser"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, HandleKey, handle)

	gotHandle, ok := GetHandleFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, handle, gotHandle)
}

func TestGetHandleFromContext_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := req.Context()

	_, ok := GetHandleFromContext(ctx)
	assert.False(t, ok)
}

func TestGetHandleFromContext_WrongType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, HandleKey, 12345)

	_, ok := GetHandleFromContext(ctx)
	assert.False(t, ok)
}
