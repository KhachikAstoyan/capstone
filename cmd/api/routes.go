package main

import (
	"net/http"

	"github.com/KhachikAstoyan/capstone/internal/api/auth"
	authhttp "github.com/KhachikAstoyan/capstone/internal/api/auth/http"
	problemshttp "github.com/KhachikAstoyan/capstone/internal/api/problems/http"
	"github.com/KhachikAstoyan/capstone/internal/api/rbac"
	rbachttp "github.com/KhachikAstoyan/capstone/internal/api/rbac/http"
	"github.com/KhachikAstoyan/capstone/pkg/permissions"
	"github.com/go-chi/chi/v5"
)

func setupRoutes(
	authHandler *authhttp.Handler,
	rbacHandler *rbachttp.Handler,
	problemsHandler *problemshttp.Handler,
	jwtManager *auth.JWTManager,
	rbacManager *rbac.Manager,
) http.Handler {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("sup"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.RefreshToken)
			r.Post("/logout", authHandler.Logout)

			r.Group(func(r chi.Router) {
				r.Use(authhttp.AuthMiddleware(jwtManager))
				r.Get("/me", authHandler.GetCurrentUser)
				r.Get("/me/roles", rbacHandler.GetMyRoles)
				r.Get("/me/permissions", rbacHandler.GetMyPermissions)
			})
		})

		// RBAC routes (protected)
		r.Group(func(r chi.Router) {
			r.Use(authhttp.AuthMiddleware(jwtManager))

			// Roles management
			r.Route("/roles", func(r chi.Router) {
				r.With(rbacManager.RequirePermission(permissions.RBACRolesView)).Get("/", rbacHandler.ListRoles)
				r.With(rbacManager.RequirePermission(permissions.RBACRolesView)).Get("/{roleID}", rbacHandler.GetRole)
				r.With(rbacManager.RequirePermission(permissions.RBACRolesView)).Get("/{roleID}/permissions", rbacHandler.GetRolePermissions)
				
				r.With(rbacManager.RequirePermission(permissions.RBACRolesManage)).Post("/", rbacHandler.CreateRole)
				r.With(rbacManager.RequirePermission(permissions.RBACRolesManage)).Put("/{roleID}", rbacHandler.UpdateRole)
				r.With(rbacManager.RequirePermission(permissions.RBACRolesManage)).Delete("/{roleID}", rbacHandler.DeleteRole)
				r.With(rbacManager.RequirePermission(permissions.RBACRolesManage)).Post("/{roleID}/permissions", rbacHandler.AssignPermissionToRole)
				r.With(rbacManager.RequirePermission(permissions.RBACRolesManage)).Delete("/{roleID}/permissions/{permissionID}", rbacHandler.RemovePermissionFromRole)
			})

			// Permissions management
			r.Route("/permissions", func(r chi.Router) {
				r.With(rbacManager.RequirePermission(permissions.RBACPermissionsView)).Get("/", rbacHandler.ListPermissions)
				r.With(rbacManager.RequirePermission(permissions.RBACPermissionsView)).Get("/{permissionID}", rbacHandler.GetPermission)
				
				r.With(rbacManager.RequirePermission(permissions.RBACPermissionsManage)).Post("/", rbacHandler.CreatePermission)
				r.With(rbacManager.RequirePermission(permissions.RBACPermissionsManage)).Put("/{permissionID}", rbacHandler.UpdatePermission)
				r.With(rbacManager.RequirePermission(permissions.RBACPermissionsManage)).Delete("/{permissionID}", rbacHandler.DeletePermission)
			})

			// User roles management
			r.Route("/users/{userID}", func(r chi.Router) {
				r.With(rbacManager.RequirePermission(permissions.RBACUsersView)).Get("/roles", rbacHandler.GetUserRoles)
				r.With(rbacManager.RequirePermission(permissions.RBACUsersView)).Get("/permissions", rbacHandler.GetUserPermissions)
				
				r.With(rbacManager.RequirePermission(permissions.RBACUsersManage)).Post("/roles", rbacHandler.AssignRoleToUser)
				r.With(rbacManager.RequirePermission(permissions.RBACUsersManage)).Delete("/roles/{roleID}", rbacHandler.RemoveRoleFromUser)
			})

			// Problems management (internal/admin only)
			r.Route("/internal/problems", func(r chi.Router) {
				r.Use(rbacManager.RequirePermission(permissions.AdminAccess))
				
				r.With(rbacManager.RequirePermission(permissions.ProblemsManage)).Post("/", problemsHandler.CreateProblem)
				r.With(rbacManager.RequirePermission(permissions.ProblemsManage)).Put("/{id}", problemsHandler.UpdateProblem)
				r.With(rbacManager.RequirePermission(permissions.ProblemsManage)).Delete("/{id}", problemsHandler.DeleteProblem)
			})
		})

		// Public problems routes
		r.Route("/problems", func(r chi.Router) {
			r.Get("/", problemsHandler.ListProblems)
			r.Get("/{id}", problemsHandler.GetProblem)
			r.Get("/slug/{slug}", problemsHandler.GetProblemBySlug)
		})
	})

	return r
}
