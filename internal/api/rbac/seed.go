package rbac

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/api/rbac/repository"
	"github.com/KhachikAstoyan/capstone/internal/api/rbac/service"
	"github.com/KhachikAstoyan/capstone/pkg/permissions"
	"github.com/google/uuid"
)

// CorePermissions defines the core RBAC permissions needed to manage the system
var CorePermissions = []struct {
	Key         string
	Description string
}{
	{permissions.RBACRolesView, "View roles and their permissions"},
	{permissions.RBACRolesManage, "Create, update, delete roles and manage role permissions"},
	{permissions.RBACPermissionsView, "View permissions"},
	{permissions.RBACPermissionsManage, "Create, update, delete permissions"},
	{permissions.RBACUsersView, "View user roles and permissions"},
	{permissions.RBACUsersManage, "Assign and remove roles from users"},
	{permissions.AdminAccess, "Access admin dashboard"},
	{permissions.ProblemsManage, "Create, update, delete problems"},
	{permissions.SubmissionsViewAll, "View any user's submissions"},
}

// CoreRoles defines the core roles for the system
var CoreRoles = []struct {
	Name        string
	Description string
	Permissions []string
}{
	{
		Name:        "super_admin",
		Description: "Super administrator with full RBAC management access",
		Permissions: []string{
			permissions.RBACRolesView,
			permissions.RBACRolesManage,
			permissions.RBACPermissionsView,
			permissions.RBACPermissionsManage,
			permissions.RBACUsersView,
			permissions.RBACUsersManage,
			permissions.AdminAccess,
			permissions.ProblemsManage,
			permissions.SubmissionsViewAll,
		},
	},
	{
		Name:        "admin",
		Description: "Administrator with user management access",
		Permissions: []string{
			permissions.RBACUsersView,
			permissions.RBACUsersManage,
		},
	},
}

// SeedCoreRBAC seeds the core RBAC permissions and roles
func SeedCoreRBAC(ctx context.Context, db *sql.DB) error {
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	userRoleRepo := repository.NewUserRoleRepository(db)
	rbacService := service.NewService(roleRepo, permRepo, userRoleRepo)

	// Create permissions
	permissionMap := make(map[string]uuid.UUID)
	for _, perm := range CorePermissions {
		// Check if permission already exists
		existing, err := permRepo.GetPermissionByKey(ctx, perm.Key)
		if err == nil {
			permissionMap[perm.Key] = existing.ID
			continue
		}

		// Create new permission
		desc := perm.Description
		created, err := rbacService.CreatePermission(ctx, perm.Key, &desc)
		if err != nil {
			return fmt.Errorf("failed to create permission %s: %w", perm.Key, err)
		}
		permissionMap[perm.Key] = created.ID
	}

	// Create roles and assign permissions
	for _, role := range CoreRoles {
		// Check if role already exists
		existing, err := roleRepo.GetRoleByName(ctx, role.Name)
		var roleID uuid.UUID
		if err == nil {
			roleID = existing.ID
		} else {
			// Create new role
			desc := role.Description
			created, err := rbacService.CreateRole(ctx, role.Name, &desc)
			if err != nil {
				return fmt.Errorf("failed to create role %s: %w", role.Name, err)
			}
			roleID = created.ID
		}

		// Assign permissions to role
		for _, permKey := range role.Permissions {
			permID, ok := permissionMap[permKey]
			if !ok {
				return fmt.Errorf("permission %s not found for role %s", permKey, role.Name)
			}

			if err := rbacService.AssignPermissionToRole(ctx, roleID, permID); err != nil {
				return fmt.Errorf("failed to assign permission %s to role %s: %w", permKey, role.Name, err)
			}
		}
	}

	return nil
}
