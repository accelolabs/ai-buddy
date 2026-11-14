package taskshttp

import (
	"context"

	"ai-buddy/internal/tracker/tasks"
)

type Service interface {
	Tasks(context.Context, string) ([]tasks.Task, error)
	Create(context.Context, string, string, tasks.CreateInput) (tasks.Task, bool, error)
	Complete(context.Context, string, string) (tasks.Task, error)
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}
