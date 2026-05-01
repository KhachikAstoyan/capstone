package rbac

import (
	"context"
	"net/http"

	authhttp "github.com/KhachikAstoyan/capstone/internal/api/auth/http"
	"github.com/KhachikAstoyan/capstone/internal/api/rbac/service"
)

// Manager provides RBAC functionality including middleware
type Manager struct {
	service service.Service
}

func NewManager(service service.Service) *Manager {
	return &Manager{
		service: service,
	}
}

// hasPermission checks if the given permissions list contains the required permission
func hasPermission(permissions []string, required string) bool {
	for _, p := range permissions {
		if p == required {
			return true
		}
	}
	return false
}

// hasAnyPermission checks if the given permissions list contains any of the required permissions
func hasAnyPermission(permissions []string, required []string) bool {
	for _, req := range required {
		if hasPermission(permissions, req) {
			return true
		}
	}
	return false
}

// hasAllPermissions checks if the given permissions list contains all of the required permissions
func hasAllPermissions(permissions []string, required []string) bool {
	for _, req := range required {
		if !hasPermission(permissions, req) {
			return false
		}
	}
	return true
}

// RequirePermission middleware checks if the authenticated user has the required permission
func (m *Manager) RequirePermission(permissionKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			permissions, ok := authhttp.GetPermissionsFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if !hasPermission(permissions, permissionKey) {
				http.Error(w, "forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission middleware checks if the user has at least one of the required permissions
func (m *Manager) RequireAnyPermission(permissionKeys ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			permissions, ok := authhttp.GetPermissionsFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if !hasAnyPermission(permissions, permissionKeys) {
				http.Error(w, "forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAllPermissions middleware checks if the user has all of the required permissions
func (m *Manager) RequireAllPermissions(permissionKeys ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			permissions, ok := authhttp.GetPermissionsFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if !hasAllPermissions(permissions, permissionKeys) {
				http.Error(w, "forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole middleware checks if the user has the required role
func (m *Manager) RequireRole(roleName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := authhttp.GetUserIDFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			hasRole, err := m.service.UserHasRole(r.Context(), userID, roleName)
			if err != nil {
				http.Error(w, "failed to check role", http.StatusInternalServerError)
				return
			}

			if !hasRole {
				http.Error(w, "forbidden: insufficient role", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CheckPermissionInContext checks if the user in context has a specific permission
func CheckPermissionInContext(ctx context.Context, permissionKey string) bool {
	permissions, ok := authhttp.GetPermissionsFromContext(ctx)
	if !ok {
		return false
	}
	return hasPermission(permissions, permissionKey)
}

// CheckAnyPermissionInContext checks if the user has any of the required permissions
func CheckAnyPermissionInContext(ctx context.Context, permissionKeys ...string) bool {
	permissions, ok := authhttp.GetPermissionsFromContext(ctx)
	if !ok {
		return false
	}
	return hasAnyPermission(permissions, permissionKeys)
}

// CheckAllPermissionsInContext checks if the user has all of the required permissions
func CheckAllPermissionsInContext(ctx context.Context, permissionKeys ...string) bool {
	permissions, ok := authhttp.GetPermissionsFromContext(ctx)
	if !ok {
		return false
	}
	return hasAllPermissions(permissions, permissionKeys)
}
