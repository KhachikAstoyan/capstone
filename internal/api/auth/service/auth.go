package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/api/auth"
	"github.com/KhachikAstoyan/capstone/internal/api/auth/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/auth/repository"
	rbacservice "github.com/KhachikAstoyan/capstone/internal/api/rbac/service"
	"github.com/KhachikAstoyan/capstone/pkg/logger"
	"github.com/KhachikAstoyan/capstone/pkg/rabbitmq"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	ProviderLocal = "local"
	PasswordAlgo  = "bcrypt"

	// emailVerificationTokenTTL is how long a link remains valid before we rotate and queue a new email.
	emailVerificationTokenTTL = 48 * time.Hour
)

// VerifyEmailOutcome is returned by VerifyEmail for HTTP messaging.
type VerifyEmailOutcome string

const (
	VerifyEmailOutcomeVerified VerifyEmailOutcome = "verified"
	VerifyEmailOutcomeResent   VerifyEmailOutcome = "resent"
)

var (
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrUserBanned               = errors.New("user is banned")
	ErrInvalidRefreshToken      = errors.New("invalid refresh token")
	ErrRevokedRefreshToken      = errors.New("refresh token has been revoked")
	ErrExpiredRefreshToken      = errors.New("refresh token has expired")
	ErrInvalidVerificationToken = errors.New("invalid or expired email verification token")
	ErrEmailNotVerified         = errors.New("email not verified")
	ErrInvalidUserRef           = errors.New("invalid user reference")
)

// RegisterSuccessMessage is returned in the register API response body.
const RegisterSuccessMessage = "Registration successful. Check your email to verify your account before signing in."

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

// RegisterResponse is returned after successful registration (no session; user must verify email, then log in).
type RegisterResponse struct {
	Message string `json:"message"`
}

type Service interface {
	Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	VerifyEmail(ctx context.Context, token string) (VerifyEmailOutcome, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	GetPublicProfile(ctx context.Context, userRef string) (*domain.PublicUserProfile, error)
	GetUserStats(ctx context.Context, userID uuid.UUID) (*domain.UserStats, error)
	ListAdminUsers(ctx context.Context, query, sortBy string, limit, offset int) ([]domain.AdminUserSummary, int, error)
	GetUserSecurityEvents(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.SecurityEvent, int, error)
}

type service struct {
	userRepo              repository.UserRepository
	identityRepo          repository.AuthIdentityRepository
	refreshTokenRepo      repository.RefreshTokenRepository
	emailVerificationRepo repository.EmailVerificationRepository
	statsRepo             repository.StatsRepository
	jwtManager            *auth.JWTManager
	rbacService           rbacservice.Service
	frontendURL           string
	emailVerificationPub  rabbitmq.EmailVerificationPublisher
}

func NewService(
	userRepo repository.UserRepository,
	identityRepo repository.AuthIdentityRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	emailVerificationRepo repository.EmailVerificationRepository,
	statsRepo repository.StatsRepository,
	jwtManager *auth.JWTManager,
	rbacService rbacservice.Service,
	frontendURL string,
	emailVerificationPub rabbitmq.EmailVerificationPublisher,
) Service {
	if emailVerificationPub == nil {
		emailVerificationPub = rabbitmq.NewNoopEmailVerificationPublisher()
	}
	return &service{
		userRepo:              userRepo,
		identityRepo:          identityRepo,
		refreshTokenRepo:      refreshTokenRepo,
		emailVerificationRepo: emailVerificationRepo,
		statsRepo:             statsRepo,
		jwtManager:            jwtManager,
		rbacService:           rbacService,
		frontendURL:           frontendURL,
		emailVerificationPub:  emailVerificationPub,
	}
}

func (s *service) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
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

	plainVerificationToken, err := auth.GenerateEmailVerificationToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate email verification token: %w", err)
	}
	verificationHash := hashToken(plainVerificationToken)
	if err := s.emailVerificationRepo.CreateEmailVerificationToken(ctx, user.ID, verificationHash); err != nil {
		return nil, fmt.Errorf("failed to store email verification token: %w", err)
	}
	verifyURL := buildEmailVerificationURL(s.frontendURL, plainVerificationToken)
	if err := s.emailVerificationPub.PublishEmailVerification(ctx, rabbitmq.EmailVerificationEvent{
		UserID:          user.ID,
		Email:           req.Email,
		VerificationURL: verifyURL,
	}); err != nil {
		logger.FromContext(ctx).Error("failed to publish email verification event",
			zap.String("user_id", user.ID.String()),
			zap.Error(err))
	}

	return &RegisterResponse{Message: RegisterSuccessMessage}, nil
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

	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
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

	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
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

func (s *service) VerifyEmail(ctx context.Context, token string) (VerifyEmailOutcome, error) {
	if token == "" {
		return "", ErrInvalidVerificationToken
	}
	tokenHash := hashToken(token)
	record, err := s.emailVerificationRepo.GetEmailVerificationTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrEmailVerificationTokenNotFound) {
			return "", ErrInvalidVerificationToken
		}
		return "", fmt.Errorf("failed to get email verification token: %w", err)
	}

	if time.Since(record.CreatedAt) > emailVerificationTokenTTL {
		if err := s.emailVerificationRepo.DeleteEmailVerificationTokenByID(ctx, record.ID); err != nil {
			return "", fmt.Errorf("failed to remove expired verification token: %w", err)
		}
		plain, err := auth.GenerateEmailVerificationToken()
		if err != nil {
			return "", fmt.Errorf("failed to generate replacement verification token: %w", err)
		}
		if err := s.emailVerificationRepo.CreateEmailVerificationToken(ctx, record.UserID, hashToken(plain)); err != nil {
			return "", fmt.Errorf("failed to store replacement verification token: %w", err)
		}
		u, err := s.userRepo.GetUserByID(ctx, record.UserID)
		if err != nil {
			return "", fmt.Errorf("failed to get user for verification email: %w", err)
		}
		verifyURL := buildEmailVerificationURL(s.frontendURL, plain)
		var emailAddr string
		if u.Email != nil {
			emailAddr = *u.Email
		}
		if err := s.emailVerificationPub.PublishEmailVerification(ctx, rabbitmq.EmailVerificationEvent{
			UserID:          record.UserID,
			Email:           emailAddr,
			VerificationURL: verifyURL,
		}); err != nil {
			logger.FromContext(ctx).Error("failed to publish email verification event (resent)",
				zap.String("user_id", record.UserID.String()),
				zap.Error(err))
		}
		return VerifyEmailOutcomeResent, nil
	}

	user, err := s.userRepo.GetUserByID(ctx, record.UserID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	if user.EmailVerified {
		if err := s.emailVerificationRepo.DeleteEmailVerificationTokenByID(ctx, record.ID); err != nil {
			return "", fmt.Errorf("failed to remove used verification token: %w", err)
		}
		return VerifyEmailOutcomeVerified, nil
	}

	if err := s.emailVerificationRepo.DeleteEmailVerificationTokenByID(ctx, record.ID); err != nil {
		return "", fmt.Errorf("failed to consume verification token: %w", err)
	}

	user.EmailVerified = true
	user.UpdatedAt = time.Now()
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	return VerifyEmailOutcomeVerified, nil
}

func (s *service) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetUserByID(ctx, userID)
}

func (s *service) GetPublicProfile(ctx context.Context, userRef string) (*domain.PublicUserProfile, error) {
	ref := strings.TrimSpace(userRef)
	if ref == "" {
		return nil, ErrInvalidUserRef
	}

	var user *domain.User
	var err error
	if id, parseErr := uuid.Parse(ref); parseErr == nil {
		user, err = s.userRepo.GetUserByID(ctx, id)
	} else {
		user, err = s.userRepo.GetUserByHandle(ctx, ref)
	}
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, repository.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.Status == domain.UserStatusBanned {
		return nil, repository.ErrUserNotFound
	}

	solved, err := s.userRepo.ListSolvedProblems(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list solved problems: %w", err)
	}
	if solved == nil {
		solved = []domain.PublicSolvedProblem{}
	}

	return &domain.PublicUserProfile{
		ID:             user.ID,
		Handle:         user.Handle,
		DisplayName:    user.DisplayName,
		AvatarURL:      user.AvatarURL,
		Status:         user.Status,
		CreatedAt:      user.CreatedAt,
		SolvedProblems: solved,
	}, nil
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

func (s *service) GetUserStats(ctx context.Context, userID uuid.UUID) (*domain.UserStats, error) {
	return s.statsRepo.GetUserStats(ctx, userID)
}

func hashToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}

func strPtr(s string) *string {
	return &s
}
