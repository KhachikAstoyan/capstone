package service

import (
	"context"

	"github.com/KhachikAstoyan/capstone/internal/api/auth/domain"
	"github.com/google/uuid"
)

func (s *service) ListAdminUsers(ctx context.Context, query, sortBy string, limit, offset int) ([]domain.AdminUserSummary, int, error) {
	return s.userRepo.ListAdminUsers(ctx, query, sortBy, limit, offset)
}

func (s *service) GetUserSecurityEvents(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.SecurityEvent, int, error) {
	return s.userRepo.GetUserSecurityEvents(ctx, userID, limit, offset)
}
