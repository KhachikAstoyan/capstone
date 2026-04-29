package http

import (
	"github.com/KhachikAstoyan/capstone/internal/api/rbac"
	"github.com/KhachikAstoyan/capstone/internal/api/submissions/service"
)

type Handler struct {
	service     service.Service
	rbacManager *rbac.Manager
}

func NewHandler(svc service.Service, rbacManager *rbac.Manager) *Handler {
	return &Handler{
		service:     svc,
		rbacManager: rbacManager,
	}
}
