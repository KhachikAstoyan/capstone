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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	RegisterFunc    func(ctx context.Context, req service.RegisterRequest) (*service.AuthResponse, error)
	LoginFunc       func(ctx context.Context, req service.LoginRequest) (*service.AuthResponse, error)
	RefreshTokenFunc func(ctx context.Context, refreshToken string) (*service.AuthResponse, error)
	LogoutFunc      func(ctx context.Context, refreshToken string) error
	GetUserByIDFunc func(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

func (m *mockService) Register(ctx context.Context, req service.RegisterRequest) (*service.AuthResponse, error) {
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

func (m *mockService) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	if m.GetUserByIDFunc != nil {
		return m.GetUserByIDFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func TestRegister_Success(t *testing.T) {
	mockSvc := &mockService{
		RegisterFunc: func(ctx context.Context, req service.RegisterRequest) (*service.AuthResponse, error) {
			return &service.AuthResponse{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				ExpiresIn:    3600,
				User: &domain.User{
					ID:     uuid.New(),
					Handle: req.Handle,
					Email:  &req.Email,
				},
			}, nil
		},
	}

	handler := NewHandler(mockSvc)

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

	var resp service.AuthResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "access-token", resp.AccessToken)
	assert.Equal(t, "refresh-token", resp.RefreshToken)
	assert.Equal(t, "testuser", resp.User.Handle)
}

func TestRegister_MissingFields(t *testing.T) {
	handler := NewHandler(&mockService{})

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
		RegisterFunc: func(ctx context.Context, req service.RegisterRequest) (*service.AuthResponse, error) {
			return nil, repository.ErrUserAlreadyExists
		},
	}

	handler := NewHandler(mockSvc)

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
		RegisterFunc: func(ctx context.Context, req service.RegisterRequest) (*service.AuthResponse, error) {
			return nil, auth.ErrPasswordTooShort
		},
	}

	handler := NewHandler(mockSvc)

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

	handler := NewHandler(mockSvc)

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

	handler := NewHandler(mockSvc)

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

	handler := NewHandler(mockSvc)

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

	handler := NewHandler(mockSvc)

	reqBody := RefreshTokenRequestDTO{
		RefreshToken: "old-refresh-token",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

	handler := NewHandler(mockSvc)

	reqBody := RefreshTokenRequestDTO{
		RefreshToken: "invalid-token",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.RefreshToken(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogout_Success(t *testing.T) {
	mockSvc := &mockService{
		LogoutFunc: func(ctx context.Context, refreshToken string) error {
			return nil
		},
	}

	handler := NewHandler(mockSvc)

	reqBody := LogoutRequestDTO{
		RefreshToken: "refresh-token",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

	handler := NewHandler(mockSvc)

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
	handler := NewHandler(&mockService{})

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

	handler := NewHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	ctx := context.WithValue(req.Context(), UserIDKey, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetCurrentUser(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func strPtr(s string) *string {
	return &s
}
