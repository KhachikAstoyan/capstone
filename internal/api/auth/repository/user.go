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
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetUserByHandle(ctx context.Context, handle string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	ListSolvedProblems(ctx context.Context, userID uuid.UUID) ([]domain.PublicSolvedProblem, error)
	UpdateUser(ctx context.Context, user *domain.User) error
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, handle, email, email_verified, display_name, avatar_url, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Handle,
		user.Email,
		user.EmailVerified,
		user.DisplayName,
		user.AvatarURL,
		user.Status,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, handle, email, email_verified, display_name, avatar_url, status, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Handle,
		&user.Email,
		&user.EmailVerified,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return user, nil
}

func (r *userRepository) GetUserByHandle(ctx context.Context, handle string) (*domain.User, error) {
	query := `
		SELECT id, handle, email, email_verified, display_name, avatar_url, status, created_at, updated_at
		FROM users
		WHERE LOWER(handle::text) = LOWER($1)
	`
	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, handle).Scan(
		&user.ID,
		&user.Handle,
		&user.Email,
		&user.EmailVerified,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by handle: %w", err)
	}
	return user, nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, handle, email, email_verified, display_name, avatar_url, status, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Handle,
		&user.Email,
		&user.EmailVerified,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

func (r *userRepository) ListSolvedProblems(ctx context.Context, userID uuid.UUID) ([]domain.PublicSolvedProblem, error) {
	query := `
		WITH latest_accepted AS (
			SELECT DISTINCT ON (s.problem_id)
				p.id AS problem_id,
				p.slug,
				p.title,
				p.summary,
				p.difficulty::text,
				s.id AS submission_id,
				s.language_id,
				l.key AS language_key,
				l.display_name AS language_display_name,
				s.source_text,
				s.status::text,
				s.created_at
			FROM submissions s
			JOIN problems p ON p.id = s.problem_id
			JOIN languages l ON l.id = s.language_id
			WHERE s.user_id = $1
			  AND s.kind = 'submit'
			  AND s.status = 'accepted'
			  AND p.visibility = 'published'
			ORDER BY s.problem_id, s.created_at DESC
		)
		SELECT problem_id, slug, title, summary, difficulty,
		       submission_id, language_id, language_key, language_display_name,
		       source_text, status, created_at
		FROM latest_accepted
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list solved problems: %w", err)
	}
	defer rows.Close()

	var solved []domain.PublicSolvedProblem
	for rows.Next() {
		var item domain.PublicSolvedProblem
		if err := rows.Scan(
			&item.ID,
			&item.Slug,
			&item.Title,
			&item.Summary,
			&item.Difficulty,
			&item.Solution.ID,
			&item.Solution.LanguageID,
			&item.Solution.LanguageKey,
			&item.Solution.LanguageDisplayName,
			&item.Solution.SourceText,
			&item.Solution.Status,
			&item.Solution.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan solved problem: %w", err)
		}
		item.SolvedAt = item.Solution.CreatedAt
		solved = append(solved, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating solved problems: %w", err)
	}
	return solved, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET handle = $2, email = $3, email_verified = $4, display_name = $5, 
		    avatar_url = $6, status = $7, updated_at = $8
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Handle,
		user.Email,
		user.EmailVerified,
		user.DisplayName,
		user.AvatarURL,
		user.Status,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}
