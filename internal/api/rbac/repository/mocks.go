package repository

import (
	"context"

	"github.com/KhachikAstoyan/capstone/internal/api/rbac/domain"
	"github.com/google/uuid"
)

type MockRoleRepository struct {
	CreateRoleFunc              func(ctx context.Context, role *domain.Role) error
	GetRoleByIDFunc             func(ctx context.Context, roleID uuid.UUID) (*domain.Role, error)
	GetRoleByNameFunc           func(ctx context.Context, name string) (*domain.Role, error)
	ListRolesFunc               func(ctx context.Context) ([]domain.Role, error)
	UpdateRoleFunc              func(ctx context.Context, role *domain.Role) error
	DeleteRoleFunc              func(ctx context.Context, roleID uuid.UUID) error
	GetRoleWithPermissionsFunc  func(ctx context.Context, roleID uuid.UUID) (*domain.RoleWithPermissions, error)
	GetRolePermissionsFunc      func(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error)
}

func (m *MockRoleRepository) CreateRole(ctx context.Context, role *domain.Role) error {
	if m.CreateRoleFunc != nil {
		return m.CreateRoleFunc(ctx, role)
	}
	return nil
}

func (m *MockRoleRepository) GetRoleByID(ctx context.Context, roleID uuid.UUID) (*domain.Role, error) {
	if m.GetRoleByIDFunc != nil {
		return m.GetRoleByIDFunc(ctx, roleID)
	}
	return nil, ErrRoleNotFound
}

func (m *MockRoleRepository) GetRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	if m.GetRoleByNameFunc != nil {
		return m.GetRoleByNameFunc(ctx, name)
	}
	return nil, ErrRoleNotFound
}

func (m *MockRoleRepository) ListRoles(ctx context.Context) ([]domain.Role, error) {
	if m.ListRolesFunc != nil {
		return m.ListRolesFunc(ctx)
	}
	return []domain.Role{}, nil
}

func (m *MockRoleRepository) UpdateRole(ctx context.Context, role *domain.Role) error {
	if m.UpdateRoleFunc != nil {
		return m.UpdateRoleFunc(ctx, role)
	}
	return nil
}

func (m *MockRoleRepository) DeleteRole(ctx context.Context, roleID uuid.UUID) error {
	if m.DeleteRoleFunc != nil {
		return m.DeleteRoleFunc(ctx, roleID)
	}
	return nil
}

func (m *MockRoleRepository) GetRoleWithPermissions(ctx context.Context, roleID uuid.UUID) (*domain.RoleWithPermissions, error) {
	if m.GetRoleWithPermissionsFunc != nil {
		return m.GetRoleWithPermissionsFunc(ctx, roleID)
	}
	return nil, ErrRoleNotFound
}

func (m *MockRoleRepository) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error) {
	if m.GetRolePermissionsFunc != nil {
		return m.GetRolePermissionsFunc(ctx, roleID)
	}
	return []domain.Permission{}, nil
}

type MockPermissionRepository struct {
	CreatePermissionFunc     func(ctx context.Context, permission *domain.Permission) error
	GetPermissionByIDFunc    func(ctx context.Context, permissionID uuid.UUID) (*domain.Permission, error)
	GetPermissionByKeyFunc   func(ctx context.Context, key string) (*domain.Permission, error)
	ListPermissionsFunc      func(ctx context.Context) ([]domain.Permission, error)
	UpdatePermissionFunc     func(ctx context.Context, permission *domain.Permission) error
	DeletePermissionFunc     func(ctx context.Context, permissionID uuid.UUID) error
}

func (m *MockPermissionRepository) CreatePermission(ctx context.Context, permission *domain.Permission) error {
	if m.CreatePermissionFunc != nil {
		return m.CreatePermissionFunc(ctx, permission)
	}
	return nil
}

func (m *MockPermissionRepository) GetPermissionByID(ctx context.Context, permissionID uuid.UUID) (*domain.Permission, error) {
	if m.GetPermissionByIDFunc != nil {
		return m.GetPermissionByIDFunc(ctx, permissionID)
	}
	return nil, ErrPermissionNotFound
}

func (m *MockPermissionRepository) GetPermissionByKey(ctx context.Context, key string) (*domain.Permission, error) {
	if m.GetPermissionByKeyFunc != nil {
		return m.GetPermissionByKeyFunc(ctx, key)
	}
	return nil, ErrPermissionNotFound
}

func (m *MockPermissionRepository) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	if m.ListPermissionsFunc != nil {
		return m.ListPermissionsFunc(ctx)
	}
	return []domain.Permission{}, nil
}

func (m *MockPermissionRepository) UpdatePermission(ctx context.Context, permission *domain.Permission) error {
	if m.UpdatePermissionFunc != nil {
		return m.UpdatePermissionFunc(ctx, permission)
	}
	return nil
}

func (m *MockPermissionRepository) DeletePermission(ctx context.Context, permissionID uuid.UUID) error {
	if m.DeletePermissionFunc != nil {
		return m.DeletePermissionFunc(ctx, permissionID)
	}
	return nil
}

type MockUserRoleRepository struct {
	AssignRoleToUserFunc           func(ctx context.Context, userRole *domain.UserRole) error
	RemoveRoleFromUserFunc         func(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRolesFunc               func(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)
	GetUserPermissionsFunc         func(ctx context.Context, userID uuid.UUID) ([]domain.Permission, error)
	AssignPermissionToRoleFunc     func(ctx context.Context, rolePermission *domain.RolePermission) error
	RemovePermissionFromRoleFunc   func(ctx context.Context, roleID, permissionID uuid.UUID) error
}

func (m *MockUserRoleRepository) AssignRoleToUser(ctx context.Context, userRole *domain.UserRole) error {
	if m.AssignRoleToUserFunc != nil {
		return m.AssignRoleToUserFunc(ctx, userRole)
	}
	return nil
}

func (m *MockUserRoleRepository) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	if m.RemoveRoleFromUserFunc != nil {
		return m.RemoveRoleFromUserFunc(ctx, userID, roleID)
	}
	return nil
}

func (m *MockUserRoleRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	if m.GetUserRolesFunc != nil {
		return m.GetUserRolesFunc(ctx, userID)
	}
	return []domain.Role{}, nil
}

func (m *MockUserRoleRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]domain.Permission, error) {
	if m.GetUserPermissionsFunc != nil {
		return m.GetUserPermissionsFunc(ctx, userID)
	}
	return []domain.Permission{}, nil
}

func (m *MockUserRoleRepository) AssignPermissionToRole(ctx context.Context, rolePermission *domain.RolePermission) error {
	if m.AssignPermissionToRoleFunc != nil {
		return m.AssignPermissionToRoleFunc(ctx, rolePermission)
	}
	return nil
}

func (m *MockUserRoleRepository) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	if m.RemovePermissionFromRoleFunc != nil {
		return m.RemovePermissionFromRoleFunc(ctx, roleID, permissionID)
	}
	return nil
}
