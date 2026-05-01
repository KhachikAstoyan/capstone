package http

import (
	"github.com/KhachikAstoyan/capstone/internal/api/problems/service"
)

type Handler struct {
	service service.Service
}

func NewHandler(svc service.Service) *Handler {
	return &Handler{
		service: svc,
	}
}
