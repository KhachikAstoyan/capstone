package repository

import (
	"context"

	"github.com/KhachikAstoyan/capstone/internal/api/auth/domain"
	"github.com/google/uuid"
)

// MockUserRepository is a mock implementation of UserRepository for testing
type MockUserRepository struct {
	CreateUserFunc         func(ctx context.Context, user *domain.User) error
	GetUserByIDFunc        func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetUserByHandleFunc    func(ctx context.Context, handle string) (*domain.User, error)
	GetUserByEmailFunc     func(ctx context.Context, email string) (*domain.User, error)
	ListSolvedProblemsFunc func(ctx context.Context, userID uuid.UUID) ([]domain.PublicSolvedProblem, error)
	UpdateUserFunc         func(ctx context.Context, user *domain.User) error
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.GetUserByIDFunc != nil {
		return m.GetUserByIDFunc(ctx, id)
	}
	return nil, ErrUserNotFound
}

func (m *MockUserRepository) GetUserByHandle(ctx context.Context, handle string) (*domain.User, error) {
	if m.GetUserByHandleFunc != nil {
		return m.GetUserByHandleFunc(ctx, handle)
	}
	return nil, ErrUserNotFound
}

func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.GetUserByEmailFunc != nil {
		return m.GetUserByEmailFunc(ctx, email)
	}
	return nil, ErrUserNotFound
}

func (m *MockUserRepository) ListSolvedProblems(ctx context.Context, userID uuid.UUID) ([]domain.PublicSolvedProblem, error) {
	if m.ListSolvedProblemsFunc != nil {
		return m.ListSolvedProblemsFunc(ctx, userID)
	}
	return []domain.PublicSolvedProblem{}, nil
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	if m.UpdateUserFunc != nil {
		return m.UpdateUserFunc(ctx, user)
	}
	return nil
}

// MockAuthIdentityRepository is a mock implementation of AuthIdentityRepository for testing
type MockAuthIdentityRepository struct {
	CreateIdentityFunc                  func(ctx context.Context, identity *domain.AuthIdentity) error
	GetIdentityByProviderAndSubjectFunc func(ctx context.Context, provider, subject string) (*domain.AuthIdentity, error)
	GetIdentitiesByUserIDFunc           func(ctx context.Context, userID uuid.UUID) ([]*domain.AuthIdentity, error)
	UpdateLastLoginFunc                 func(ctx context.Context, identityID uuid.UUID) error
}

func (m *MockAuthIdentityRepository) CreateIdentity(ctx context.Context, identity *domain.AuthIdentity) error {
	if m.CreateIdentityFunc != nil {
		return m.CreateIdentityFunc(ctx, identity)
	}
	return nil
}

func (m *MockAuthIdentityRepository) GetIdentityByProviderAndSubject(ctx context.Context, provider, subject string) (*domain.AuthIdentity, error) {
	if m.GetIdentityByProviderAndSubjectFunc != nil {
		return m.GetIdentityByProviderAndSubjectFunc(ctx, provider, subject)
	}
	return nil, ErrIdentityNotFound
}

func (m *MockAuthIdentityRepository) GetIdentitiesByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.AuthIdentity, error) {
	if m.GetIdentitiesByUserIDFunc != nil {
		return m.GetIdentitiesByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockAuthIdentityRepository) UpdateLastLogin(ctx context.Context, identityID uuid.UUID) error {
	if m.UpdateLastLoginFunc != nil {
		return m.UpdateLastLoginFunc(ctx, identityID)
	}
	return nil
}

// MockRefreshTokenRepository is a mock implementation of RefreshTokenRepository for testing
type MockRefreshTokenRepository struct {
	CreateRefreshTokenFunc    func(ctx context.Context, token *domain.RefreshToken) error
	GetRefreshTokenByHashFunc func(ctx context.Context, tokenHash []byte) (*domain.RefreshToken, error)
	RevokeRefreshTokenFunc    func(ctx context.Context, tokenID uuid.UUID, replacedBy *uuid.UUID) error
	RevokeAllUserTokensFunc   func(ctx context.Context, userID uuid.UUID) error
	CleanupExpiredTokensFunc  func(ctx context.Context) error
}

func (m *MockRefreshTokenRepository) CreateRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	if m.CreateRefreshTokenFunc != nil {
		return m.CreateRefreshTokenFunc(ctx, token)
	}
	return nil
}

func (m *MockRefreshTokenRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash []byte) (*domain.RefreshToken, error) {
	if m.GetRefreshTokenByHashFunc != nil {
		return m.GetRefreshTokenByHashFunc(ctx, tokenHash)
	}
	return nil, ErrRefreshTokenNotFound
}

func (m *MockRefreshTokenRepository) RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID, replacedBy *uuid.UUID) error {
	if m.RevokeRefreshTokenFunc != nil {
		return m.RevokeRefreshTokenFunc(ctx, tokenID, replacedBy)
	}
	return nil
}

func (m *MockRefreshTokenRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	if m.RevokeAllUserTokensFunc != nil {
		return m.RevokeAllUserTokensFunc(ctx, userID)
	}
	return nil
}

func (m *MockRefreshTokenRepository) CleanupExpiredTokens(ctx context.Context) error {
	if m.CleanupExpiredTokensFunc != nil {
		return m.CleanupExpiredTokensFunc(ctx)
	}
	return nil
}

// MockEmailVerificationRepository is a mock implementation of EmailVerificationRepository for testing.
type MockEmailVerificationRepository struct {
	CreateEmailVerificationTokenFunc     func(ctx context.Context, userID uuid.UUID, tokenHash []byte) error
	GetEmailVerificationTokenByHashFunc  func(ctx context.Context, tokenHash []byte) (*domain.EmailVerificationToken, error)
	DeleteEmailVerificationTokenByIDFunc func(ctx context.Context, id uuid.UUID) error
}

func (m *MockEmailVerificationRepository) CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash []byte) error {
	if m.CreateEmailVerificationTokenFunc != nil {
		return m.CreateEmailVerificationTokenFunc(ctx, userID, tokenHash)
	}
	return nil
}

func (m *MockEmailVerificationRepository) GetEmailVerificationTokenByHash(ctx context.Context, tokenHash []byte) (*domain.EmailVerificationToken, error) {
	if m.GetEmailVerificationTokenByHashFunc != nil {
		return m.GetEmailVerificationTokenByHashFunc(ctx, tokenHash)
	}
	return nil, ErrEmailVerificationTokenNotFound
}

func (m *MockEmailVerificationRepository) DeleteEmailVerificationTokenByID(ctx context.Context, id uuid.UUID) error {
	if m.DeleteEmailVerificationTokenByIDFunc != nil {
		return m.DeleteEmailVerificationTokenByIDFunc(ctx, id)
	}
	return nil
}
