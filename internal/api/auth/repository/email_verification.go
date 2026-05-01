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
	ErrEmailVerificationTokenNotFound = errors.New("email verification token not found")
)

type EmailVerificationRepository interface {
	// GetEmailVerificationTokenByHash returns the row for tokenHash without deleting it.
	// ErrEmailVerificationTokenNotFound if no row matched.
	GetEmailVerificationTokenByHash(ctx context.Context, tokenHash []byte) (*domain.EmailVerificationToken, error)
	DeleteEmailVerificationTokenByID(ctx context.Context, id uuid.UUID) error
	CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash []byte) error
}

type emailVerificationRepository struct {
	db *sql.DB
}

func NewEmailVerificationRepository(db *sql.DB) EmailVerificationRepository {
	return &emailVerificationRepository{db: db}
}

func (r *emailVerificationRepository) GetEmailVerificationTokenByHash(ctx context.Context, tokenHash []byte) (*domain.EmailVerificationToken, error) {
	query := `
		SELECT id, user_id, token_hash, created_at
		FROM email_verification_tokens
		WHERE token_hash = $1
	`
	token := &domain.EmailVerificationToken{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEmailVerificationTokenNotFound
		}
		return nil, fmt.Errorf("failed to get email verification token: %w", err)
	}
	return token, nil
}

func (r *emailVerificationRepository) DeleteEmailVerificationTokenByID(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM email_verification_tokens WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete email verification token: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read rows affected: %w", err)
	}
	if n == 0 {
		return ErrEmailVerificationTokenNotFound
	}
	return nil
}

func (r *emailVerificationRepository) CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash []byte) error {
	query := `
		INSERT INTO email_verification_tokens (user_id, token_hash)
		VALUES ($1, $2)
	`

	_, err := r.db.ExecContext(ctx, query, userID, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to create email verification token: %w", err)
	}

	return nil
}
