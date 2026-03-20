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
	ErrRoleNotFound      = errors.New("role not found")
	ErrRoleAlreadyExists = errors.New("role already exists")
)

type RoleRepository interface {
	CreateRole(ctx context.Context, role *domain.Role) error
	GetRoleByID(ctx context.Context, id uuid.UUID) (*domain.Role, error)
	GetRoleByName(ctx context.Context, name string) (*domain.Role, error)
	ListRoles(ctx context.Context) ([]domain.Role, error)
	UpdateRole(ctx context.Context, role *domain.Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error
	GetRoleWithPermissions(ctx context.Context, roleID uuid.UUID) (*domain.RoleWithPermissions, error)
}

type roleRepository struct {
	db *sql.DB
}

func NewRoleRepository(db *sql.DB) RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) CreateRole(ctx context.Context, role *domain.Role) error {
	query := `
		INSERT INTO roles (id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query,
		role.ID,
		role.Name,
		role.Description,
		role.CreatedAt,
		role.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrRoleAlreadyExists
		}
		return fmt.Errorf("failed to create role: %w", err)
	}
	return nil
}

func (r *roleRepository) GetRoleByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		WHERE id = $1
	`
	role := &domain.Role{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to get role by id: %w", err)
	}
	return role, nil
}

func (r *roleRepository) GetRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		WHERE name = $1
	`
	role := &domain.Role{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to get role by name: %w", err)
	}
	return role, nil
}

func (r *roleRepository) ListRoles(ctx context.Context) ([]domain.Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		ORDER BY name
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var role domain.Role
		err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.Description,
			&role.CreatedAt,
			&role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return roles, nil
}

func (r *roleRepository) UpdateRole(ctx context.Context, role *domain.Role) error {
	query := `
		UPDATE roles
		SET name = $2, description = $3, updated_at = $4
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query,
		role.ID,
		role.Name,
		role.Description,
		role.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrRoleAlreadyExists
		}
		return fmt.Errorf("failed to update role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrRoleNotFound
	}

	return nil
}

func (r *roleRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM roles WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrRoleNotFound
	}

	return nil
}

func (r *roleRepository) GetRoleWithPermissions(ctx context.Context, roleID uuid.UUID) (*domain.RoleWithPermissions, error) {
	role, err := r.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT p.id, p.key, p.description, p.created_at, p.updated_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.key
	`
	rows, err := r.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
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

	return &domain.RoleWithPermissions{
		Role:        *role,
		Permissions: permissions,
	}, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && (
		err.Error() == "pq: duplicate key value violates unique constraint" ||
		err.Error() == "UNIQUE constraint failed")
}
