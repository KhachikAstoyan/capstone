package service

import (
	"context"

	"github.com/KhachikAstoyan/capstone/internal/api/rbac/domain"
	"github.com/google/uuid"
)

type MockService struct {
	CreateRoleFunc               func(ctx context.Context, name string, description *string) (*domain.Role, error)
	GetRoleFunc                  func(ctx context.Context, roleID uuid.UUID) (*domain.Role, error)
	GetRoleByNameFunc            func(ctx context.Context, name string) (*domain.Role, error)
	ListRolesFunc                func(ctx context.Context) ([]domain.Role, error)
	UpdateRoleFunc               func(ctx context.Context, roleID uuid.UUID, name string, description *string) (*domain.Role, error)
	DeleteRoleFunc               func(ctx context.Context, roleID uuid.UUID) error
	GetRoleWithPermissionsFunc   func(ctx context.Context, roleID uuid.UUID) (*domain.RoleWithPermissions, error)
	GetRolePermissionsFunc       func(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error)
	CreatePermissionFunc         func(ctx context.Context, key string, description *string) (*domain.Permission, error)
	GetPermissionFunc            func(ctx context.Context, permissionID uuid.UUID) (*domain.Permission, error)
	GetPermissionByKeyFunc       func(ctx context.Context, key string) (*domain.Permission, error)
	ListPermissionsFunc          func(ctx context.Context) ([]domain.Permission, error)
	UpdatePermissionFunc         func(ctx context.Context, permissionID uuid.UUID, key string, description *string) (*domain.Permission, error)
	DeletePermissionFunc         func(ctx context.Context, permissionID uuid.UUID) error
	AssignRoleToUserFunc         func(ctx context.Context, userID, roleID uuid.UUID, grantedBy *uuid.UUID) error
	RemoveRoleFromUserFunc       func(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRolesFunc             func(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)
	GetUserPermissionsFunc       func(ctx context.Context, userID uuid.UUID) ([]domain.Permission, error)
	GetUserWithRolesFunc         func(ctx context.Context, userID uuid.UUID) (*domain.UserWithRoles, error)
	AssignPermissionToRoleFunc   func(ctx context.Context, roleID, permissionID uuid.UUID) error
	RemovePermissionFromRoleFunc func(ctx context.Context, roleID, permissionID uuid.UUID) error
	UserHasPermissionFunc        func(ctx context.Context, userID uuid.UUID, permissionKey string) (bool, error)
	UserHasRoleFunc              func(ctx context.Context, userID uuid.UUID, roleName string) (bool, error)
}

func (m *MockService) CreateRole(ctx context.Context, name string, description *string) (*domain.Role, error) {
	if m.CreateRoleFunc != nil {
		return m.CreateRoleFunc(ctx, name, description)
	}
	return &domain.Role{ID: uuid.New(), Name: name, Description: description}, nil
}

func (m *MockService) GetRole(ctx context.Context, roleID uuid.UUID) (*domain.Role, error) {
	if m.GetRoleFunc != nil {
		return m.GetRoleFunc(ctx, roleID)
	}
	return nil, nil
}

func (m *MockService) GetRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	if m.GetRoleByNameFunc != nil {
		return m.GetRoleByNameFunc(ctx, name)
	}
	return nil, nil
}

func (m *MockService) ListRoles(ctx context.Context) ([]domain.Role, error) {
	if m.ListRolesFunc != nil {
		return m.ListRolesFunc(ctx)
	}
	return []domain.Role{}, nil
}

func (m *MockService) UpdateRole(ctx context.Context, roleID uuid.UUID, name string, description *string) (*domain.Role, error) {
	if m.UpdateRoleFunc != nil {
		return m.UpdateRoleFunc(ctx, roleID, name, description)
	}
	return &domain.Role{ID: roleID, Name: name, Description: description}, nil
}

func (m *MockService) DeleteRole(ctx context.Context, roleID uuid.UUID) error {
	if m.DeleteRoleFunc != nil {
		return m.DeleteRoleFunc(ctx, roleID)
	}
	return nil
}

func (m *MockService) GetRoleWithPermissions(ctx context.Context, roleID uuid.UUID) (*domain.RoleWithPermissions, error) {
	if m.GetRoleWithPermissionsFunc != nil {
		return m.GetRoleWithPermissionsFunc(ctx, roleID)
	}
	return nil, nil
}

func (m *MockService) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error) {
	if m.GetRolePermissionsFunc != nil {
		return m.GetRolePermissionsFunc(ctx, roleID)
	}
	return []domain.Permission{}, nil
}

func (m *MockService) CreatePermission(ctx context.Context, key string, description *string) (*domain.Permission, error) {
	if m.CreatePermissionFunc != nil {
		return m.CreatePermissionFunc(ctx, key, description)
	}
	return &domain.Permission{ID: uuid.New(), Key: key, Description: description}, nil
}

func (m *MockService) GetPermission(ctx context.Context, permissionID uuid.UUID) (*domain.Permission, error) {
	if m.GetPermissionFunc != nil {
		return m.GetPermissionFunc(ctx, permissionID)
	}
	return nil, nil
}

func (m *MockService) GetPermissionByKey(ctx context.Context, key string) (*domain.Permission, error) {
	if m.GetPermissionByKeyFunc != nil {
		return m.GetPermissionByKeyFunc(ctx, key)
	}
	return nil, nil
}

func (m *MockService) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	if m.ListPermissionsFunc != nil {
		return m.ListPermissionsFunc(ctx)
	}
	return []domain.Permission{}, nil
}

func (m *MockService) UpdatePermission(ctx context.Context, permissionID uuid.UUID, key string, description *string) (*domain.Permission, error) {
	if m.UpdatePermissionFunc != nil {
		return m.UpdatePermissionFunc(ctx, permissionID, key, description)
	}
	return &domain.Permission{ID: permissionID, Key: key, Description: description}, nil
}

func (m *MockService) DeletePermission(ctx context.Context, permissionID uuid.UUID) error {
	if m.DeletePermissionFunc != nil {
		return m.DeletePermissionFunc(ctx, permissionID)
	}
	return nil
}

func (m *MockService) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID, grantedBy *uuid.UUID) error {
	if m.AssignRoleToUserFunc != nil {
		return m.AssignRoleToUserFunc(ctx, userID, roleID, grantedBy)
	}
	return nil
}

func (m *MockService) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	if m.RemoveRoleFromUserFunc != nil {
		return m.RemoveRoleFromUserFunc(ctx, userID, roleID)
	}
	return nil
}

func (m *MockService) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	if m.GetUserRolesFunc != nil {
		return m.GetUserRolesFunc(ctx, userID)
	}
	return []domain.Role{}, nil
}

func (m *MockService) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]domain.Permission, error) {
	if m.GetUserPermissionsFunc != nil {
		return m.GetUserPermissionsFunc(ctx, userID)
	}
	return []domain.Permission{}, nil
}

func (m *MockService) GetUserWithRoles(ctx context.Context, userID uuid.UUID) (*domain.UserWithRoles, error) {
	if m.GetUserWithRolesFunc != nil {
		return m.GetUserWithRolesFunc(ctx, userID)
	}
	return &domain.UserWithRoles{
		UserID:      userID,
		Roles:       []domain.Role{},
		Permissions: []domain.Permission{},
	}, nil
}

func (m *MockService) AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	if m.AssignPermissionToRoleFunc != nil {
		return m.AssignPermissionToRoleFunc(ctx, roleID, permissionID)
	}
	return nil
}

func (m *MockService) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	if m.RemovePermissionFromRoleFunc != nil {
		return m.RemovePermissionFromRoleFunc(ctx, roleID, permissionID)
	}
	return nil
}

func (m *MockService) UserHasPermission(ctx context.Context, userID uuid.UUID, permissionKey string) (bool, error) {
	if m.UserHasPermissionFunc != nil {
		return m.UserHasPermissionFunc(ctx, userID, permissionKey)
	}
	return false, nil
}

func (m *MockService) UserHasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error) {
	if m.UserHasRoleFunc != nil {
		return m.UserHasRoleFunc(ctx, userID, roleName)
	}
	return false, nil
}
