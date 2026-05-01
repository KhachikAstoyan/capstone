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
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

type RefreshTokenRepository interface {
	CreateRefreshToken(ctx context.Context, token *domain.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash []byte) (*domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID, replacedBy *uuid.UUID) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error
	CleanupExpiredTokens(ctx context.Context) error
}

type refreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) CreateRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, auth_identity_id, token_hash, issued_at, expires_at, revoked_at, replaced_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		token.ID,
		token.UserID,
		token.AuthIdentityID,
		token.TokenHash,
		token.IssuedAt,
		token.ExpiresAt,
		token.RevokedAt,
		token.ReplacedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash []byte) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, auth_identity_id, token_hash, issued_at, expires_at, revoked_at, replaced_by
		FROM refresh_tokens
		WHERE token_hash = $1
	`
	token := &domain.RefreshToken{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.AuthIdentityID,
		&token.TokenHash,
		&token.IssuedAt,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.ReplacedBy,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	return token, nil
}

func (r *refreshTokenRepository) RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID, replacedBy *uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), replaced_by = $2
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, tokenID, replacedBy)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all user tokens: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) CleanupExpiredTokens(ctx context.Context) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE expires_at < NOW() - INTERVAL '30 days'
	`
	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && (err.Error() == "pq: duplicate key value violates unique constraint" ||
		err.Error() == "UNIQUE constraint failed")
}
