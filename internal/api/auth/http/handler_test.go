package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KhachikAstoyan/capstone/internal/api/auth"
	"github.com/KhachikAstoyan/capstone/internal/api/auth/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/auth/repository"
	"github.com/KhachikAstoyan/capstone/internal/api/auth/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	RegisterFunc     func(ctx context.Context, req service.RegisterRequest) (*service.RegisterResponse, error)
	LoginFunc        func(ctx context.Context, req service.LoginRequest) (*service.AuthResponse, error)
	RefreshTokenFunc func(ctx context.Context, refreshToken string) (*service.AuthResponse, error)
	LogoutFunc       func(ctx context.Context, refreshToken string) error
	VerifyEmailFunc func(ctx context.Context, token string) (service.VerifyEmailOutcome, error)
	GetUserByIDFunc        func(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	GetPublicProfileFunc   func(ctx context.Context, userRef string) (*domain.PublicUserProfile, error)
}

func (m *mockService) Register(ctx context.Context, req service.RegisterRequest) (*service.RegisterResponse, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockService) Login(ctx context.Context, req service.LoginRequest) (*service.AuthResponse, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockService) RefreshToken(ctx context.Context, refreshToken string) (*service.AuthResponse, error) {
	if m.RefreshTokenFunc != nil {
		return m.RefreshTokenFunc(ctx, refreshToken)
	}
	return nil, errors.New("not implemented")
}

func (m *mockService) Logout(ctx context.Context, refreshToken string) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(ctx, refreshToken)
	}
	return errors.New("not implemented")
}

func (m *mockService) VerifyEmail(ctx context.Context, token string) (service.VerifyEmailOutcome, error) {
	if m.VerifyEmailFunc != nil {
		return m.VerifyEmailFunc(ctx, token)
	}
	return "", errors.New("not implemented")
}

func (m *mockService) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	if m.GetUserByIDFunc != nil {
		return m.GetUserByIDFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockService) GetPublicProfile(ctx context.Context, userRef string) (*domain.PublicUserProfile, error) {
	if m.GetPublicProfileFunc != nil {
		return m.GetPublicProfileFunc(ctx, userRef)
	}
	return nil, errors.New("not implemented")
}

func TestRegister_Success(t *testing.T) {
	mockSvc := &mockService{
		RegisterFunc: func(ctx context.Context, req service.RegisterRequest) (*service.RegisterResponse, error) {
			return &service.RegisterResponse{Message: "ok"}, nil
		},
	}

	handler := NewHandler(mockSvc, false)

	reqBody := RegisterRequestDTO{
		Handle:   "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp service.RegisterResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Message)
}

func TestRegister_MissingFields(t *testing.T) {
	handler := NewHandler(&mockService{}, false)

	tests := []struct {
		name string
		body RegisterRequestDTO
	}{
		{
			name: "missing handle",
			body: RegisterRequestDTO{
				Email:    "test@example.com",
				Password: "password123",
			},
		},
		{
			name: "missing email",
			body: RegisterRequestDTO{
				Handle:   "testuser",
				Password: "password123",
			},
		},
		{
			name: "missing password",
			body: RegisterRequestDTO{
				Handle: "testuser",
				Email:  "test@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Register(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestRegister_UserAlreadyExists(t *testing.T) {
	mockSvc := &mockService{
		RegisterFunc: func(ctx context.Context, req service.RegisterRequest) (*service.RegisterResponse, error) {
			return nil, repository.ErrUserAlreadyExists
		},
	}

	handler := NewHandler(mockSvc, false)

	reqBody := RegisterRequestDTO{
		Handle:   "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRegister_PasswordTooShort(t *testing.T) {
	mockSvc := &mockService{
		RegisterFunc: func(ctx context.Context, req service.RegisterRequest) (*service.RegisterResponse, error) {
			return nil, auth.ErrPasswordTooShort
		},
	}

	handler := NewHandler(mockSvc, false)

	reqBody := RegisterRequestDTO{
		Handle:   "testuser",
		Email:    "test@example.com",
		Password: "short",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_Success(t *testing.T) {
	mockSvc := &mockService{
		LoginFunc: func(ctx context.Context, req service.LoginRequest) (*service.AuthResponse, error) {
			return &service.AuthResponse{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				ExpiresIn:    3600,
				User: &domain.User{
					ID:     uuid.New(),
					Handle: "testuser",
					Email:  &req.Email,
				},
			}, nil
		},
	}

	handler := NewHandler(mockSvc, false)

	reqBody := LoginRequestDTO{
		Email:    "test@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp service.AuthResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "access-token", resp.AccessToken)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	mockSvc := &mockService{
		LoginFunc: func(ctx context.Context, req service.LoginRequest) (*service.AuthResponse, error) {
			return nil, service.ErrInvalidCredentials
		},
	}

	handler := NewHandler(mockSvc, false)

	reqBody := LoginRequestDTO{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_UserBanned(t *testing.T) {
	mockSvc := &mockService{
		LoginFunc: func(ctx context.Context, req service.LoginRequest) (*service.AuthResponse, error) {
			return nil, service.ErrUserBanned
		},
	}

	handler := NewHandler(mockSvc, false)

	reqBody := LoginRequestDTO{
		Email:    "test@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestLogin_EmailNotVerified(t *testing.T) {
	mockSvc := &mockService{
		LoginFunc: func(ctx context.Context, req service.LoginRequest) (*service.AuthResponse, error) {
			return nil, service.ErrEmailNotVerified
		},
	}

	handler := NewHandler(mockSvc, false)

	reqBody := LoginRequestDTO{
		Email:    "test@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRefreshToken_Success(t *testing.T) {
	mockSvc := &mockService{
		RefreshTokenFunc: func(ctx context.Context, refreshToken string) (*service.AuthResponse, error) {
			return &service.AuthResponse{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
				ExpiresIn:    3600,
				User: &domain.User{
					ID:     uuid.New(),
					Handle: "testuser",
				},
			}, nil
		},
	}

	handler := NewHandler(mockSvc, false)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "old-refresh-token",
	})
	w := httptest.NewRecorder()

	handler.RefreshToken(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp service.AuthResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", resp.AccessToken)
	assert.Equal(t, "new-refresh-token", resp.RefreshToken)
}

func TestRefreshToken_Invalid(t *testing.T) {
	mockSvc := &mockService{
		RefreshTokenFunc: func(ctx context.Context, refreshToken string) (*service.AuthResponse, error) {
			return nil, service.ErrInvalidRefreshToken
		},
	}

	handler := NewHandler(mockSvc, false)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "invalid-token",
	})
	w := httptest.NewRecorder()

	handler.RefreshToken(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefreshToken_EmailNotVerified(t *testing.T) {
	mockSvc := &mockService{
		RefreshTokenFunc: func(ctx context.Context, refreshToken string) (*service.AuthResponse, error) {
			return nil, service.ErrEmailNotVerified
		},
	}

	handler := NewHandler(mockSvc, false)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "refresh-token",
	})
	w := httptest.NewRecorder()

	handler.RefreshToken(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestLogout_Success(t *testing.T) {
	mockSvc := &mockService{
		LogoutFunc: func(ctx context.Context, refreshToken string) error {
			return nil
		},
	}

	handler := NewHandler(mockSvc, false)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "refresh-token",
	})
	w := httptest.NewRecorder()

	handler.Logout(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetCurrentUser_Success(t *testing.T) {
	userID := uuid.New()
	mockSvc := &mockService{
		GetUserByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			assert.Equal(t, userID, id)
			return &domain.User{
				ID:     userID,
				Handle: "testuser",
				Email:  strPtr("test@example.com"),
			}, nil
		},
	}

	handler := NewHandler(mockSvc, false)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	ctx := context.WithValue(req.Context(), UserIDKey, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetCurrentUser(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var user domain.User
	err := json.NewDecoder(w.Body).Decode(&user)
	require.NoError(t, err)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "testuser", user.Handle)
}

func TestGetCurrentUser_NotAuthenticated(t *testing.T) {
	handler := NewHandler(&mockService{}, false)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	w := httptest.NewRecorder()

	handler.GetCurrentUser(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetCurrentUser_UserNotFound(t *testing.T) {
	userID := uuid.New()
	mockSvc := &mockService{
		GetUserByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return nil, repository.ErrUserNotFound
		},
	}

	handler := NewHandler(mockSvc, false)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	ctx := context.WithValue(req.Context(), UserIDKey, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetCurrentUser(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetPublicUserProfile_ByHandle(t *testing.T) {
	mockSvc := &mockService{
		GetPublicProfileFunc: func(ctx context.Context, userRef string) (*domain.PublicUserProfile, error) {
			assert.Equal(t, "johndoe", userRef)
			return &domain.PublicUserProfile{
				ID:     uuid.New(),
				Handle: "johndoe",
			}, nil
		},
	}

	handler := NewHandler(mockSvc, false)
	r := chi.NewRouter()
	r.Get("/users/{userRef}", handler.GetPublicUserProfile)

	req := httptest.NewRequest(http.MethodGet, "/users/johndoe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetPublicUserProfile_NotFound(t *testing.T) {
	mockSvc := &mockService{
		GetPublicProfileFunc: func(ctx context.Context, userRef string) (*domain.PublicUserProfile, error) {
			return nil, repository.ErrUserNotFound
		},
	}

	handler := NewHandler(mockSvc, false)
	r := chi.NewRouter()
	r.Get("/users/{userRef}", handler.GetPublicUserProfile)

	req := httptest.NewRequest(http.MethodGet, "/users/nobody", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func strPtr(s string) *string {
	return &s
}
