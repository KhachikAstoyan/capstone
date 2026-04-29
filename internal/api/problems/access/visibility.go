package access

import (
	"context"

	authhttp "github.com/KhachikAstoyan/capstone/internal/api/auth/http"
	"github.com/KhachikAstoyan/capstone/internal/api/problems/domain"
	"github.com/KhachikAstoyan/capstone/pkg/permissions"
)

// CanViewUnpublishedProblems is true when the caller has the same rights as problem admins
// (draft/archived listings and detail).
func CanViewUnpublishedProblems(ctx context.Context) bool {
	perms, ok := authhttp.GetPermissionsFromContext(ctx)
	if !ok {
		return false
	}
	var hasAdmin, hasManage bool
	for _, p := range perms {
		switch p {
		case permissions.AdminAccess:
			hasAdmin = true
		case permissions.ProblemsManage:
			hasManage = true
		}
	}
	return hasAdmin && hasManage
}

// IsPublished reports whether the problem is visible to anonymous users and the public catalog.
func IsPublished(p *domain.Problem) bool {
	return p != nil && p.Visibility == domain.VisibilityPublished
}
