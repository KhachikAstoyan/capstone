package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/api/rbac/domain"
	"github.com/google/uuid"
)

type UserRoleRepository interface {
	AssignRoleToUser(ctx context.Context, userRole *domain.UserRole) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]domain.Permission, error)
	GetUserWithRoles(ctx context.Context, userID uuid.UUID) (*domain.UserWithRoles, error)
	AssignPermissionToRole(ctx context.Context, rolePermission *domain.RolePermission) error
	RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error
}

type userRoleRepository struct {
	db *sql.DB
}

func NewUserRoleRepository(db *sql.DB) UserRoleRepository {
	return &userRoleRepository{db: db}
}

func (r *userRoleRepository) AssignRoleToUser(ctx context.Context, userRole *domain.UserRole) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, granted_by, granted_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query,
		userRole.UserID,
		userRole.RoleID,
		userRole.GrantedBy,
		userRole.GrantedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to assign role to user: %w", err)
	}
	return nil
}

func (r *userRoleRepository) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	query := `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`
	_, err := r.db.ExecContext(ctx, query, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to remove role from user: %w", err)
	}
	return nil
}

func (r *userRoleRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.created_at, r.updated_at
		FROM roles r
		INNER JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY r.name
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	roles := []domain.Role{}
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

func (r *userRoleRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]domain.Permission, error) {
	query := `
		SELECT DISTINCT p.id, p.key, p.description, p.created_at, p.updated_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		INNER JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.key
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}
	defer rows.Close()

	permissions := []domain.Permission{}
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

func (r *userRoleRepository) GetUserWithRoles(ctx context.Context, userID uuid.UUID) (*domain.UserWithRoles, error) {
	roles, err := r.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	permissions, err := r.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &domain.UserWithRoles{
		UserID:      userID,
		Roles:       roles,
		Permissions: permissions,
	}, nil
}

func (r *userRoleRepository) AssignPermissionToRole(ctx context.Context, rolePermission *domain.RolePermission) error {
	query := `
		INSERT INTO role_permissions (role_id, permission_id)
		VALUES ($1, $2)
		ON CONFLICT (role_id, permission_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query,
		rolePermission.RoleID,
		rolePermission.PermissionID,
	)
	if err != nil {
		return fmt.Errorf("failed to assign permission to role: %w", err)
	}
	return nil
}

func (r *userRoleRepository) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`
	_, err := r.db.ExecContext(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to remove permission from role: %w", err)
	}
	return nil
}
