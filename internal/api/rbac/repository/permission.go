package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/api/rbac/domain"
	"github.com/google/uuid"
)

var (
	ErrPermissionNotFound      = errors.New("permission not found")
	ErrPermissionAlreadyExists = errors.New("permission already exists")
)

type PermissionRepository interface {
	CreatePermission(ctx context.Context, permission *domain.Permission) error
	GetPermissionByID(ctx context.Context, id uuid.UUID) (*domain.Permission, error)
	GetPermissionByKey(ctx context.Context, key string) (*domain.Permission, error)
	ListPermissions(ctx context.Context) ([]domain.Permission, error)
	UpdatePermission(ctx context.Context, permission *domain.Permission) error
	DeletePermission(ctx context.Context, id uuid.UUID) error
}

type permissionRepository struct {
	db *sql.DB
}

func NewPermissionRepository(db *sql.DB) PermissionRepository {
	return &permissionRepository{db: db}
}

func (r *permissionRepository) CreatePermission(ctx context.Context, permission *domain.Permission) error {
	query := `
		INSERT INTO permissions (id, key, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query,
		permission.ID,
		permission.Key,
		permission.Description,
		permission.CreatedAt,
		permission.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrPermissionAlreadyExists
		}
		return fmt.Errorf("failed to create permission: %w", err)
	}
	return nil
}

func (r *permissionRepository) GetPermissionByID(ctx context.Context, id uuid.UUID) (*domain.Permission, error) {
	query := `
		SELECT id, key, description, created_at, updated_at
		FROM permissions
		WHERE id = $1
	`
	permission := &domain.Permission{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&permission.ID,
		&permission.Key,
		&permission.Description,
		&permission.CreatedAt,
		&permission.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPermissionNotFound
		}
		return nil, fmt.Errorf("failed to get permission by id: %w", err)
	}
	return permission, nil
}

func (r *permissionRepository) GetPermissionByKey(ctx context.Context, key string) (*domain.Permission, error) {
	query := `
		SELECT id, key, description, created_at, updated_at
		FROM permissions
		WHERE key = $1
	`
	permission := &domain.Permission{}
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&permission.ID,
		&permission.Key,
		&permission.Description,
		&permission.CreatedAt,
		&permission.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPermissionNotFound
		}
		return nil, fmt.Errorf("failed to get permission by key: %w", err)
	}
	return permission, nil
}

func (r *permissionRepository) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	query := `
		SELECT id, key, description, created_at, updated_at
		FROM permissions
		ORDER BY key
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	defer rows.Close()

	var permissions []domain.Permission
	for rows.Next() {
		var perm domain.Permission
		err := rows.Scan(
			&perm.ID,
			&perm.Key,
			&perm.Description,
			&perm.CreatedAt,
			&perm.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, perm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return permissions, nil
}

func (r *permissionRepository) UpdatePermission(ctx context.Context, permission *domain.Permission) error {
	query := `
		UPDATE permissions
		SET key = $2, description = $3, updated_at = $4
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		permission.ID,
		permission.Key,
		permission.Description,
		permission.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrPermissionAlreadyExists
		}
		return fmt.Errorf("failed to update permission: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrPermissionNotFound
	}

	return nil
}

func (r *permissionRepository) DeletePermission(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM permissions WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrPermissionNotFound
	}

	return nil
}
