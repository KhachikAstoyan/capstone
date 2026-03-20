package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/api/auth"
	"github.com/KhachikAstoyan/capstone/internal/api/auth/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/auth/repository"
	rbacservice "github.com/KhachikAstoyan/capstone/internal/api/rbac/service"
	"github.com/google/uuid"
)

const (
	ProviderLocal = "local"
	PasswordAlgo  = "bcrypt"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserBanned          = errors.New("user is banned")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRevokedRefreshToken = errors.New("refresh token has been revoked")
	ErrExpiredRefreshToken = errors.New("refresh token has expired")
)

type RegisterRequest struct {
	Handle      string
	Email       string
	Password    string
	DisplayName *string
}

type LoginRequest struct {
	Email    string
	Password string
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
	User         *domain.User `json:"user"`
}

type Service interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

type service struct {
	userRepo         repository.UserRepository
	identityRepo     repository.AuthIdentityRepository
	refreshTokenRepo repository.RefreshTokenRepository
	jwtManager       *auth.JWTManager
	rbacService      rbacservice.Service
}

func NewService(
	userRepo repository.UserRepository,
	identityRepo repository.AuthIdentityRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtManager *auth.JWTManager,
	rbacService rbacservice.Service,
) Service {
	return &service{
		userRepo:         userRepo,
		identityRepo:     identityRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtManager:       jwtManager,
		rbacService:      rbacService,
	}
}

func (s *service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	existingUser, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, repository.ErrUserAlreadyExists
	}

	existingUser, err = s.userRepo.GetUserByHandle(ctx, req.Handle)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing handle: %w", err)
	}
	if existingUser != nil {
		return nil, repository.ErrUserAlreadyExists
	}

	passwordHash, err := auth.HashAndValidatePassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	user := &domain.User{
		ID:            uuid.New(),
		Handle:        req.Handle,
		Email:         &req.Email,
		EmailVerified: false,
		DisplayName:   req.DisplayName,
		Status:        domain.UserStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	identity := &domain.AuthIdentity{
		ID:              uuid.New(),
		UserID:          user.ID,
		Provider:        ProviderLocal,
		ProviderSubject: req.Email,
		PasswordHash:    &passwordHash,
		PasswordAlgo:    strPtr(PasswordAlgo),
		EmailAtProvider: &req.Email,
		CreatedAt:       now,
	}

	if err := s.identityRepo.CreateIdentity(ctx, identity); err != nil {
		return nil, fmt.Errorf("failed to create identity: %w", err)
	}

	return s.generateAuthResponse(ctx, user, identity.ID)
}

func (s *service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	identity, err := s.identityRepo.GetIdentityByProviderAndSubject(ctx, ProviderLocal, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrIdentityNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}

	if identity.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}

	if err := auth.ComparePassword(*identity.PasswordHash, req.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.userRepo.GetUserByID(ctx, identity.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.Status == domain.UserStatusBanned {
		return nil, ErrUserBanned
	}

	if err := s.identityRepo.UpdateLastLogin(ctx, identity.ID); err != nil {
		return nil, fmt.Errorf("failed to update last login: %w", err)
	}

	return s.generateAuthResponse(ctx, user, identity.ID)
}

func (s *service) RefreshToken(ctx context.Context, refreshTokenStr string) (*AuthResponse, error) {
	tokenHash := hashToken(refreshTokenStr)

	storedToken, err := s.refreshTokenRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	if storedToken.RevokedAt != nil {
		return nil, ErrRevokedRefreshToken
	}

	if time.Now().After(storedToken.ExpiresAt) {
		return nil, ErrExpiredRefreshToken
	}

	user, err := s.userRepo.GetUserByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.Status == domain.UserStatusBanned {
		return nil, ErrUserBanned
	}

	newRefreshTokenStr, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	newTokenHash := hashToken(newRefreshTokenStr)
	now := time.Now()
	newToken := &domain.RefreshToken{
		ID:             uuid.New(),
		UserID:         user.ID,
		AuthIdentityID: storedToken.AuthIdentityID,
		TokenHash:      newTokenHash,
		IssuedAt:       now,
		ExpiresAt:      now.Add(s.jwtManager.GetRefreshTokenDuration()),
	}

	if err := s.refreshTokenRepo.CreateRefreshToken(ctx, newToken); err != nil {
		return nil, fmt.Errorf("failed to create new refresh token: %w", err)
	}

	if err := s.refreshTokenRepo.RevokeRefreshToken(ctx, storedToken.ID, &newToken.ID); err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	// Fetch user permissions for the token
	var permissionKeys []string
	userWithRoles, err := s.rbacService.GetUserWithRoles(ctx, user.ID)
	if err != nil {
		// If RBAC fails, continue with empty permissions
		permissionKeys = []string{}
	} else {
		permissionKeys = userWithRoles.GetPermissionKeys()
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Handle, permissionKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshTokenStr,
		ExpiresIn:    int64(s.jwtManager.GetRefreshTokenDuration().Seconds()),
		User:         user,
	}, nil
}

func (s *service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)

	storedToken, err := s.refreshTokenRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			return nil
		}
		return fmt.Errorf("failed to get refresh token: %w", err)
	}

	if err := s.refreshTokenRepo.RevokeRefreshToken(ctx, storedToken.ID, nil); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

func (s *service) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetUserByID(ctx, userID)
}

func (s *service) generateAuthResponse(ctx context.Context, user *domain.User, identityID uuid.UUID) (*AuthResponse, error) {
	// Fetch user permissions for the token
	var permissionKeys []string
	userWithRoles, err := s.rbacService.GetUserWithRoles(ctx, user.ID)
	if err != nil {
		// If RBAC fails, continue with empty permissions
		// This prevents auth from breaking if RBAC has issues
		permissionKeys = []string{}
	} else {
		permissionKeys = userWithRoles.GetPermissionKeys()
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Handle, permissionKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	tokenHash := hashToken(refreshToken)
	now := time.Now()
	refreshTokenRecord := &domain.RefreshToken{
		ID:             uuid.New(),
		UserID:         user.ID,
		AuthIdentityID: &identityID,
		TokenHash:      tokenHash,
		IssuedAt:       now,
		ExpiresAt:      now.Add(s.jwtManager.GetRefreshTokenDuration()),
	}

	if err := s.refreshTokenRepo.CreateRefreshToken(ctx, refreshTokenRecord); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtManager.GetRefreshTokenDuration().Seconds()),
		User:         user,
	}, nil
}

func hashToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}

func strPtr(s string) *string {
	return &s
}
