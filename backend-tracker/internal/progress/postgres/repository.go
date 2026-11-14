package postgres

import (
	"context"

	"ai-buddy/backend-tracker/internal/interns"
	"ai-buddy/backend-tracker/internal/progress"
	"ai-buddy/backend-tracker/internal/tasks"
)

// These interfaces belong to this adapter, which consumes the SQL repositories.
type InternRepository interface {
	Intern(context.Context, string) (interns.Profile, error)
	Mentor(context.Context, string) (*interns.Profile, error)
	Assigned(context.Context, string, string) (bool, error)
}

type TaskRepository interface {
	Tasks(context.Context, string) ([]tasks.Task, error)
}

type Repository struct {
	interns InternRepository
	tasks   TaskRepository
}

func New(interns InternRepository, tasks TaskRepository) *Repository {
	return &Repository{interns: interns, tasks: tasks}
}

func (r *Repository) Load(ctx context.Context, id string) (progress.Snapshot, error) {
	intern, err := r.interns.Intern(ctx, id)
	if err != nil {
		return progress.Snapshot{}, err
	}

	mentor, err := r.interns.Mentor(ctx, id)
	if err != nil {
		return progress.Snapshot{}, err
	}

	list, err := r.tasks.Tasks(ctx, id)
	if err != nil {
		return progress.Snapshot{}, err
	}

	return progress.Snapshot{
		Intern: intern,
		Mentor: mentor,
		Tasks:  list,
	}, nil
}

func (r *Repository) Assigned(ctx context.Context, mentor, id string) (bool, error) {
	return r.interns.Assigned(ctx, mentor, id)
}
