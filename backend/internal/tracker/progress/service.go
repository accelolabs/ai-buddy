package progress

import (
	"context"
	"time"

	"ai-buddy/internal/tracker/apperror"
	"ai-buddy/internal/tracker/interns"
	"ai-buddy/internal/tracker/tasks"
)

type InternRepository interface {
	Intern(context.Context, string) (interns.Profile, error)
	Mentor(context.Context, string) (*interns.Profile, error)
	Assigned(context.Context, string, string) (bool, error)
}

type TaskRepository interface {
	// Tasks returns a fresh caller-owned slice ordered by created_at, then ID.
	Tasks(context.Context, string) ([]tasks.Task, error)
}

type Service struct {
	interns  InternRepository
	tasks    TaskRepository
	location *time.Location
	now      func() time.Time
}

func New(interns InternRepository, tasks TaskRepository, location *time.Location, now func() time.Time) *Service {
	return &Service{
		interns:  interns,
		tasks:    tasks,
		location: location,
		now:      now,
	}
}

func (s *Service) Progress(ctx context.Context, id string) (Progress, error) {
	intern, err := s.interns.Intern(ctx, id)
	if err != nil {
		return Progress{}, err
	}

	mentor, err := s.interns.Mentor(ctx, id)
	if err != nil {
		return Progress{}, err
	}
	raw, err := s.tasks.Tasks(ctx, id)
	if err != nil {
		return Progress{}, err
	}
	list := tasks.PrepareList(raw, tasks.CalendarDate(s.now(), s.location))
	result := Progress{
		Intern:       intern,
		Mentor:       mentor,
		StartDate:    intern.StartDate,
		Goals:        intern.Goals,
		Expectations: intern.Expectations,
		Tasks:        list,
		Total:        len(list),
	}
	for i := range list {
		task := &list[i]
		if task.CompletedAt != nil {
			result.Completed++
			continue
		}

		next := result.NextTask
		if next == nil || (!next.Overdue && (task.Overdue || task.Deadline < next.Deadline)) {
			result.NextTask = task
		}
	}

	if result.Total > 0 {
		result.Percent = 100 * result.Completed / result.Total
	}

	return result, nil
}

func (s *Service) MentorProgress(ctx context.Context, mentor, id string) (Progress, error) {
	ok, err := s.interns.Assigned(ctx, mentor, id)
	if err != nil {
		return Progress{}, err
	}

	if !ok {
		return Progress{}, apperror.ErrNotFound
	}

	return s.Progress(ctx, id)
}
