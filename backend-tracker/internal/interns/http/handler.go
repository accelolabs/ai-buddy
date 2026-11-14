package internshttp

import (
	"context"

	"ai-buddy/backend-tracker/internal/interns"
)

type Service interface {
	Interns(context.Context, string) ([]interns.Profile, error)
	Mentor(context.Context, string) (*interns.Profile, error)
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}
