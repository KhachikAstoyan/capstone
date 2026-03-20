package service

import (
	"context"
	"testing"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/api/auth"
	"github.com/KhachikAstoyan/capstone/internal/api/auth/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/auth/repository"
	rbacservice "github.com/KhachikAstoyan/capstone/internal/api/rbac/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestService() (*service, *repository.MockUserRepository, *repository.MockAuthIdentityRepository, *repository.MockRefreshTokenRepository, *rbacservice.MockService) {
	userRepo := &repository.MockUserRepository{}
	identityRepo := &repository.MockAuthIdentityRepository{}
	tokenRepo := &repository.MockRefreshTokenRepository{}
	rbacSvc := &rbacservice.MockService{}
	jwtManager := auth.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	svc := &service{
		userRepo:         userRepo,
		identityRepo:     identityRepo,
		refreshTokenRepo: tokenRepo,
		jwtManager:       jwtManager,
		rbacService:      rbacSvc,
	}

	return svc, userRepo, identityRepo, tokenRepo, rbacSvc
}

func TestRegister_Success(t *testing.T) {
	svc, userRepo, identityRepo, tokenRepo, _ := setupTestService()
	ctx := context.Background()

	userRepo.GetUserByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
		return nil, repository.ErrUserNotFound
	}

	userRepo.GetUserByHandleFunc = func(ctx context.Context, handle string) (*domain.User, error) {
		return nil, repository.ErrUserNotFound
	}

	userRepo.CreateUserFunc = func(ctx context.Context, user *domain.User) error {
		assert.Equal(t, "testuser", user.Handle)
		assert.Equal(t, "test@example.com", *user.Email)
		assert.Equal(t, domain.UserStatusActive, user.Status)
		return nil
	}

	identityRepo.CreateIdentityFunc = func(ctx context.Context, identity *domain.AuthIdentity) error {
		assert.Equal(t, ProviderLocal, identity.Provider)
		assert.NotNil(t, identity.PasswordHash)
		return nil
	}

	tokenRepo.CreateRefreshTokenFunc = func(ctx context.Context, token *domain.RefreshToken) error {
		assert.NotNil(t, token.TokenHash)
		return nil
	}

	req := RegisterRequest{
		Handle:   "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	resp, err := svc.Register(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.NotNil(t, resp.User)
	assert.Equal(t, "testuser", resp.User.Handle)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc, userRepo, _, _, _ := setupTestService()
	ctx := context.Background()

	existingUser := &domain.User{
		ID:     uuid.New(),
		Handle: "existing",
		Email:  strPtr("test@example.com"),
	}

	userRepo.GetUserByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
		return existingUser, nil
	}

	req := RegisterRequest{
		Handle:   "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	_, err := svc.Register(ctx, req)
	assert.ErrorIs(t, err, repository.ErrUserAlreadyExists)
}

func TestRegister_DuplicateHandle(t *testing.T) {
	svc, userRepo, _, _, _ := setupTestService()
	ctx := context.Background()

	userRepo.GetUserByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
		return nil, repository.ErrUserNotFound
	}

	existingUser := &domain.User{
		ID:     uuid.New(),
		Handle: "testuser",
	}

	userRepo.GetUserByHandleFunc = func(ctx context.Context, handle string) (*domain.User, error) {
		return existingUser, nil
	}

	req := RegisterRequest{
		Handle:   "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	_, err := svc.Register(ctx, req)
	assert.ErrorIs(t, err, repository.ErrUserAlreadyExists)
}

func TestRegister_WeakPassword(t *testing.T) {
	svc, userRepo, _, _, _ := setupTestService()
	ctx := context.Background()

	userRepo.GetUserByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
		return nil, repository.ErrUserNotFound
	}

	userRepo.GetUserByHandleFunc = func(ctx context.Context, handle string) (*domain.User, error) {
		return nil, repository.ErrUserNotFound
	}

	req := RegisterRequest{
		Handle:   "testuser",
		Email:    "test@example.com",
		Password: "weak",
	}

	_, err := svc.Register(ctx, req)
	assert.Error(t, err)
}

func TestLogin_Success(t *testing.T) {
	svc, userRepo, identityRepo, tokenRepo, _ := setupTestService()
	ctx := context.Background()

	password := "password123"
	passwordHash, err := auth.HashAndValidatePassword(password)
	require.NoError(t, err)

	userID := uuid.New()
	identityID := uuid.New()

	identityRepo.GetIdentityByProviderAndSubjectFunc = func(ctx context.Context, provider, subject string) (*domain.AuthIdentity, error) {
		return &domain.AuthIdentity{
			ID:           identityID,
			UserID:       userID,
			Provider:     ProviderLocal,
			PasswordHash: &passwordHash,
		}, nil
	}

	userRepo.GetUserByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
		return &domain.User{
			ID:     userID,
			Handle: "testuser",
			Email:  strPtr("test@example.com"),
			Status: domain.UserStatusActive,
		}, nil
	}

	identityRepo.UpdateLastLoginFunc = func(ctx context.Context, id uuid.UUID) error {
		assert.Equal(t, identityID, id)
		return nil
	}

	tokenRepo.CreateRefreshTokenFunc = func(ctx context.Context, token *domain.RefreshToken) error {
		return nil
	}

	req := LoginRequest{
		Email:    "test@example.com",
		Password: password,
	}

	resp, err := svc.Login(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.NotNil(t, resp.User)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	svc, _, identityRepo, _, _ := setupTestService()
	ctx := context.Background()

	identityRepo.GetIdentityByProviderAndSubjectFunc = func(ctx context.Context, provider, subject string) (*domain.AuthIdentity, error) {
		return nil, repository.ErrIdentityNotFound
	}

	req := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	_, err := svc.Login(ctx, req)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, _, identityRepo, _, _ := setupTestService()
	ctx := context.Background()

	passwordHash, err := auth.HashAndValidatePassword("correctPassword")
	require.NoError(t, err)

	identityRepo.GetIdentityByProviderAndSubjectFunc = func(ctx context.Context, provider, subject string) (*domain.AuthIdentity, error) {
		return &domain.AuthIdentity{
			ID:           uuid.New(),
			UserID:       uuid.New(),
			PasswordHash: &passwordHash,
		}, nil
	}

	req := LoginRequest{
		Email:    "test@example.com",
		Password: "wrongPassword",
	}

	_, err = svc.Login(ctx, req)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_BannedUser(t *testing.T) {
	svc, userRepo, identityRepo, _, _ := setupTestService()
	ctx := context.Background()

	password := "password123"
	passwordHash, err := auth.HashAndValidatePassword(password)
	require.NoError(t, err)

	userID := uuid.New()

	identityRepo.GetIdentityByProviderAndSubjectFunc = func(ctx context.Context, provider, subject string) (*domain.AuthIdentity, error) {
		return &domain.AuthIdentity{
			ID:           uuid.New(),
			UserID:       userID,
			PasswordHash: &passwordHash,
		}, nil
	}

	userRepo.GetUserByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
		return &domain.User{
			ID:     userID,
			Status: domain.UserStatusBanned,
		}, nil
	}

	req := LoginRequest{
		Email:    "test@example.com",
		Password: password,
	}

	_, err = svc.Login(ctx, req)
	assert.ErrorIs(t, err, ErrUserBanned)
}

func TestRefreshToken_Success(t *testing.T) {
	svc, userRepo, _, tokenRepo, _ := setupTestService()
	ctx := context.Background()

	userID := uuid.New()
	refreshToken := "valid-refresh-token"
	tokenHash := hashToken(refreshToken)

	tokenRepo.GetRefreshTokenByHashFunc = func(ctx context.Context, hash []byte) (*domain.RefreshToken, error) {
		assert.Equal(t, tokenHash, hash)
		return &domain.RefreshToken{
			ID:        uuid.New(),
			UserID:    userID,
			TokenHash: hash,
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}, nil
	}

	userRepo.GetUserByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
		return &domain.User{
			ID:     userID,
			Handle: "testuser",
			Status: domain.UserStatusActive,
		}, nil
	}

	tokenRepo.CreateRefreshTokenFunc = func(ctx context.Context, token *domain.RefreshToken) error {
		return nil
	}

	tokenRepo.RevokeRefreshTokenFunc = func(ctx context.Context, tokenID uuid.UUID, replacedBy *uuid.UUID) error {
		assert.NotNil(t, replacedBy)
		return nil
	}

	resp, err := svc.RefreshToken(ctx, refreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.NotEqual(t, refreshToken, resp.RefreshToken)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	svc, _, _, tokenRepo, _ := setupTestService()
	ctx := context.Background()

	tokenRepo.GetRefreshTokenByHashFunc = func(ctx context.Context, hash []byte) (*domain.RefreshToken, error) {
		return nil, repository.ErrRefreshTokenNotFound
	}

	_, err := svc.RefreshToken(ctx, "invalid-token")
	assert.ErrorIs(t, err, ErrInvalidRefreshToken)
}

func TestRefreshToken_RevokedToken(t *testing.T) {
	svc, _, _, tokenRepo, _ := setupTestService()
	ctx := context.Background()

	now := time.Now()
	tokenRepo.GetRefreshTokenByHashFunc = func(ctx context.Context, hash []byte) (*domain.RefreshToken, error) {
		return &domain.RefreshToken{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			RevokedAt: &now,
		}, nil
	}

	_, err := svc.RefreshToken(ctx, "revoked-token")
	assert.ErrorIs(t, err, ErrRevokedRefreshToken)
}

func TestRefreshToken_ExpiredToken(t *testing.T) {
	svc, _, _, tokenRepo, _ := setupTestService()
	ctx := context.Background()

	tokenRepo.GetRefreshTokenByHashFunc = func(ctx context.Context, hash []byte) (*domain.RefreshToken, error) {
		return &domain.RefreshToken{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}, nil
	}

	_, err := svc.RefreshToken(ctx, "expired-token")
	assert.ErrorIs(t, err, ErrExpiredRefreshToken)
}

func TestLogout_Success(t *testing.T) {
	svc, _, _, tokenRepo, _ := setupTestService()
	ctx := context.Background()

	tokenID := uuid.New()
	tokenRepo.GetRefreshTokenByHashFunc = func(ctx context.Context, hash []byte) (*domain.RefreshToken, error) {
		return &domain.RefreshToken{
			ID:     tokenID,
			UserID: uuid.New(),
		}, nil
	}

	tokenRepo.RevokeRefreshTokenFunc = func(ctx context.Context, id uuid.UUID, replacedBy *uuid.UUID) error {
		assert.Equal(t, tokenID, id)
		assert.Nil(t, replacedBy)
		return nil
	}

	err := svc.Logout(ctx, "valid-token")
	assert.NoError(t, err)
}

func TestLogout_TokenNotFound(t *testing.T) {
	svc, _, _, tokenRepo, _ := setupTestService()
	ctx := context.Background()

	tokenRepo.GetRefreshTokenByHashFunc = func(ctx context.Context, hash []byte) (*domain.RefreshToken, error) {
		return nil, repository.ErrRefreshTokenNotFound
	}

	err := svc.Logout(ctx, "invalid-token")
	assert.NoError(t, err) // Should not error on missing token
}

func TestGetUserByID(t *testing.T) {
	svc, userRepo, _, _, _ := setupTestService()
	ctx := context.Background()

	userID := uuid.New()
	expectedUser := &domain.User{
		ID:     userID,
		Handle: "testuser",
	}

	userRepo.GetUserByIDFunc = func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
		assert.Equal(t, userID, id)
		return expectedUser, nil
	}

	user, err := svc.GetUserByID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, expectedUser, user)
}
