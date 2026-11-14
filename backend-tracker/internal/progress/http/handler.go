package progresshttp

import (
	"context"

	"ai-buddy/backend-tracker/internal/progress"
)

type Service interface {
	Progress(context.Context, string) (progress.Progress, error)
	MentorProgress(context.Context, string, string) (progress.Progress, error)
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}
