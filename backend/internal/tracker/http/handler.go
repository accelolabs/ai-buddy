package trackerhttp

import (
	"context"

	"ai-buddy/internal/tracker/interns"
	"ai-buddy/internal/tracker/progress"
	"ai-buddy/internal/tracker/tasks"
)

type TaskService interface {
	Tasks(context.Context, string) ([]tasks.Task, error)
	Create(context.Context, string, string, tasks.CreateInput) (tasks.Task, bool, error)
	Complete(context.Context, string, string) (tasks.Task, error)
}

type ProfileReader interface {
	Interns(context.Context, string) ([]interns.Profile, error)
}

type ProgressService interface {
	Progress(context.Context, string) (progress.Progress, error)
	MentorProgress(context.Context, string, string) (progress.Progress, error)
}

type Handler struct {
	tasks    TaskService
	profiles ProfileReader
	progress ProgressService
}

func New(tasks TaskService, profiles ProfileReader, progress ProgressService) *Handler {
	return &Handler{tasks: tasks, profiles: profiles, progress: progress}
}
