package service

import (
	"context"
	"fmt"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/api/rbac/domain"
	"github.com/KhachikAstoyan/capstone/internal/api/rbac/repository"
	"github.com/google/uuid"
)

type Service interface {
	// Role management
	CreateRole(ctx context.Context, name string, description *string) (*domain.Role, error)
	GetRole(ctx context.Context, roleID uuid.UUID) (*domain.Role, error)
	GetRoleByName(ctx context.Context, name string) (*domain.Role, error)
	ListRoles(ctx context.Context) ([]domain.Role, error)
	UpdateRole(ctx context.Context, roleID uuid.UUID, name string, description *string) (*domain.Role, error)
	DeleteRole(ctx context.Context, roleID uuid.UUID) error
	GetRoleWithPermissions(ctx context.Context, roleID uuid.UUID) (*domain.RoleWithPermissions, error)

	// Permission management
	CreatePermission(ctx context.Context, key string, description *string) (*domain.Permission, error)
	GetPermission(ctx context.Context, permissionID uuid.UUID) (*domain.Permission, error)
	GetPermissionByKey(ctx context.Context, key string) (*domain.Permission, error)
	ListPermissions(ctx context.Context) ([]domain.Permission, error)
	UpdatePermission(ctx context.Context, permissionID uuid.UUID, key string, description *string) (*domain.Permission, error)
	DeletePermission(ctx context.Context, permissionID uuid.UUID) error

	// User-Role assignments
	AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID, grantedBy *uuid.UUID) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]domain.Permission, error)
	GetUserWithRoles(ctx context.Context, userID uuid.UUID) (*domain.UserWithRoles, error)

	// Role-Permission assignments
	AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error

	// Permission checking
	UserHasPermission(ctx context.Context, userID uuid.UUID, permissionKey string) (bool, error)
	UserHasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error)
}

type service struct {
	roleRepo     repository.RoleRepository
	permRepo     repository.PermissionRepository
	userRoleRepo repository.UserRoleRepository
}

func NewService(
	roleRepo repository.RoleRepository,
	permRepo repository.PermissionRepository,
	userRoleRepo repository.UserRoleRepository,
) Service {
	return &service{
		roleRepo:     roleRepo,
		permRepo:     permRepo,
		userRoleRepo: userRoleRepo,
	}
}

// Role management

func (s *service) CreateRole(ctx context.Context, name string, description *string) (*domain.Role, error) {
	now := time.Now()
	role := &domain.Role{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.roleRepo.CreateRole(ctx, role); err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return role, nil
}

func (s *service) GetRole(ctx context.Context, roleID uuid.UUID) (*domain.Role, error) {
	return s.roleRepo.GetRoleByID(ctx, roleID)
}

func (s *service) GetRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	return s.roleRepo.GetRoleByName(ctx, name)
}

func (s *service) ListRoles(ctx context.Context) ([]domain.Role, error) {
	return s.roleRepo.ListRoles(ctx)
}

func (s *service) UpdateRole(ctx context.Context, roleID uuid.UUID, name string, description *string) (*domain.Role, error) {
	role, err := s.roleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}

	role.Name = name
	role.Description = description
	role.UpdatedAt = time.Now()

	if err := s.roleRepo.UpdateRole(ctx, role); err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return role, nil
}

func (s *service) DeleteRole(ctx context.Context, roleID uuid.UUID) error {
	return s.roleRepo.DeleteRole(ctx, roleID)
}

func (s *service) GetRoleWithPermissions(ctx context.Context, roleID uuid.UUID) (*domain.RoleWithPermissions, error) {
	return s.roleRepo.GetRoleWithPermissions(ctx, roleID)
}

// Permission management

func (s *service) CreatePermission(ctx context.Context, key string, description *string) (*domain.Permission, error) {
	now := time.Now()
	permission := &domain.Permission{
		ID:          uuid.New(),
		Key:         key,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.permRepo.CreatePermission(ctx, permission); err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	return permission, nil
}

func (s *service) GetPermission(ctx context.Context, permissionID uuid.UUID) (*domain.Permission, error) {
	return s.permRepo.GetPermissionByID(ctx, permissionID)
}

func (s *service) GetPermissionByKey(ctx context.Context, key string) (*domain.Permission, error) {
	return s.permRepo.GetPermissionByKey(ctx, key)
}

func (s *service) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	return s.permRepo.ListPermissions(ctx)
}

func (s *service) UpdatePermission(ctx context.Context, permissionID uuid.UUID, key string, description *string) (*domain.Permission, error) {
	permission, err := s.permRepo.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return nil, err
	}

	permission.Key = key
	permission.Description = description
	permission.UpdatedAt = time.Now()

	if err := s.permRepo.UpdatePermission(ctx, permission); err != nil {
		return nil, fmt.Errorf("failed to update permission: %w", err)
	}

	return permission, nil
}

func (s *service) DeletePermission(ctx context.Context, permissionID uuid.UUID) error {
	return s.permRepo.DeletePermission(ctx, permissionID)
}

// User-Role assignments

func (s *service) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID, grantedBy *uuid.UUID) error {
	userRole := &domain.UserRole{
		UserID:    userID,
		RoleID:    roleID,
		GrantedBy: grantedBy,
		GrantedAt: time.Now(),
	}

	return s.userRoleRepo.AssignRoleToUser(ctx, userRole)
}

func (s *service) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return s.userRoleRepo.RemoveRoleFromUser(ctx, userID, roleID)
}

func (s *service) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	return s.userRoleRepo.GetUserRoles(ctx, userID)
}

func (s *service) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]domain.Permission, error) {
	return s.userRoleRepo.GetUserPermissions(ctx, userID)
}

func (s *service) GetUserWithRoles(ctx context.Context, userID uuid.UUID) (*domain.UserWithRoles, error) {
	return s.userRoleRepo.GetUserWithRoles(ctx, userID)
}

// Role-Permission assignments

func (s *service) AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	rolePermission := &domain.RolePermission{
		RoleID:       roleID,
		PermissionID: permissionID,
	}

	return s.userRoleRepo.AssignPermissionToRole(ctx, rolePermission)
}

func (s *service) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	return s.userRoleRepo.RemovePermissionFromRole(ctx, roleID, permissionID)
}

// Permission checking

func (s *service) UserHasPermission(ctx context.Context, userID uuid.UUID, permissionKey string) (bool, error) {
	userWithRoles, err := s.userRoleRepo.GetUserWithRoles(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user roles: %w", err)
	}

	return userWithRoles.HasPermission(permissionKey), nil
}

func (s *service) UserHasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error) {
	userWithRoles, err := s.userRoleRepo.GetUserWithRoles(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user roles: %w", err)
	}

	return userWithRoles.HasRole(roleName), nil
}
