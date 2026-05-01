package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Permission struct {
	ID          uuid.UUID `json:"id"`
	Key         string    `json:"key"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserRole struct {
	UserID    uuid.UUID  `json:"user_id"`
	RoleID    uuid.UUID  `json:"role_id"`
	GrantedBy *uuid.UUID `json:"granted_by,omitempty"`
	GrantedAt time.Time  `json:"granted_at"`
}

type RolePermission struct {
	RoleID       uuid.UUID `json:"role_id"`
	PermissionID uuid.UUID `json:"permission_id"`
}

// RoleWithPermissions represents a role with its associated permissions
type RoleWithPermissions struct {
	Role        Role         `json:"role"`
	Permissions []Permission `json:"permissions"`
}

// UserWithRoles represents a user with their roles and permissions
type UserWithRoles struct {
	UserID      uuid.UUID             `json:"user_id"`
	Roles       []Role                `json:"roles"`
	Permissions []Permission          `json:"permissions"`
	RoleMap     map[string]Role       `json:"-"`
	PermMap     map[string]Permission `json:"-"`
}

func (u *UserWithRoles) HasRole(roleName string) bool {
	if u.RoleMap == nil {
		u.buildMaps()
	}
	_, exists := u.RoleMap[roleName]
	return exists
}

func (u *UserWithRoles) HasPermission(permKey string) bool {
	if u.PermMap == nil {
		u.buildMaps()
	}
	_, exists := u.PermMap[permKey]
	return exists
}

func (u *UserWithRoles) HasAnyPermission(permKeys ...string) bool {
	if u.PermMap == nil {
		u.buildMaps()
	}
	for _, key := range permKeys {
		if _, exists := u.PermMap[key]; exists {
			return true
		}
	}
	return false
}

func (u *UserWithRoles) HasAllPermissions(permKeys ...string) bool {
	if u.PermMap == nil {
		u.buildMaps()
	}
	for _, key := range permKeys {
		if _, exists := u.PermMap[key]; !exists {
			return false
		}
	}
	return true
}

func (u *UserWithRoles) buildMaps() {
	u.RoleMap = make(map[string]Role)
	u.PermMap = make(map[string]Permission)

	for _, role := range u.Roles {
		u.RoleMap[role.Name] = role
	}

	for _, perm := range u.Permissions {
		u.PermMap[perm.Key] = perm
	}
}

// GetPermissionKeys returns a slice of all permission keys the user has
func (u *UserWithRoles) GetPermissionKeys() []string {
	keys := make([]string, 0, len(u.Permissions))
	for _, perm := range u.Permissions {
		keys = append(keys, perm.Key)
	}
	return keys
}
