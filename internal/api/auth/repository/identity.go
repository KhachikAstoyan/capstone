package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/api/auth/domain"
	"github.com/google/uuid"
)

var (
	ErrIdentityNotFound = errors.New("auth identity not found")
)

type AuthIdentityRepository interface {
	CreateIdentity(ctx context.Context, identity *domain.AuthIdentity) error
	GetIdentityByProviderAndSubject(ctx context.Context, provider, subject string) (*domain.AuthIdentity, error)
	GetIdentitiesByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.AuthIdentity, error)
	UpdateLastLogin(ctx context.Context, identityID uuid.UUID) error
}

type authIdentityRepository struct {
	db *sql.DB
}

func NewAuthIdentityRepository(db *sql.DB) AuthIdentityRepository {
	return &authIdentityRepository{db: db}
}

func (r *authIdentityRepository) CreateIdentity(ctx context.Context, identity *domain.AuthIdentity) error {
	query := `
		INSERT INTO auth_identities (
			id, user_id, provider, provider_subject, password_hash, password_algo,
			email_at_provider, email_verified_at_provider, created_at, last_login_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		identity.ID,
		identity.UserID,
		identity.Provider,
		identity.ProviderSubject,
		identity.PasswordHash,
		identity.PasswordAlgo,
		identity.EmailAtProvider,
		identity.EmailVerifiedAtProvider,
		identity.CreatedAt,
		identity.LastLoginAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create auth identity: %w", err)
	}
	return nil
}

func (r *authIdentityRepository) GetIdentityByProviderAndSubject(ctx context.Context, provider, subject string) (*domain.AuthIdentity, error) {
	query := `
		SELECT id, user_id, provider, provider_subject, password_hash, password_algo,
		       email_at_provider, email_verified_at_provider, created_at, last_login_at
		FROM auth_identities
		WHERE provider = $1 AND provider_subject = $2
	`
	identity := &domain.AuthIdentity{}
	err := r.db.QueryRowContext(ctx, query, provider, subject).Scan(
		&identity.ID,
		&identity.UserID,
		&identity.Provider,
		&identity.ProviderSubject,
		&identity.PasswordHash,
		&identity.PasswordAlgo,
		&identity.EmailAtProvider,
		&identity.EmailVerifiedAtProvider,
		&identity.CreatedAt,
		&identity.LastLoginAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrIdentityNotFound
		}
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}
	return identity, nil
}

func (r *authIdentityRepository) GetIdentitiesByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.AuthIdentity, error) {
	query := `
		SELECT id, user_id, provider, provider_subject, password_hash, password_algo,
		       email_at_provider, email_verified_at_provider, created_at, last_login_at
		FROM auth_identities
		WHERE user_id = $1
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get identities: %w", err)
	}
	defer rows.Close()

	var identities []*domain.AuthIdentity
	for rows.Next() {
		identity := &domain.AuthIdentity{}
		err := rows.Scan(
			&identity.ID,
			&identity.UserID,
			&identity.Provider,
			&identity.ProviderSubject,
			&identity.PasswordHash,
			&identity.PasswordAlgo,
			&identity.EmailAtProvider,
			&identity.EmailVerifiedAtProvider,
			&identity.CreatedAt,
			&identity.LastLoginAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan identity: %w", err)
		}
		identities = append(identities, identity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return identities, nil
}

func (r *authIdentityRepository) UpdateLastLogin(ctx context.Context, identityID uuid.UUID) error {
	query := `
		UPDATE auth_identities
		SET last_login_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, identityID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}
