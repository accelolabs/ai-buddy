package progress

import (
	"context"
	"time"

	"ai-buddy/backend-tracker/internal/apperror"
	"ai-buddy/backend-tracker/internal/tasks"
)

type Repository interface {
	Load(context.Context, string) (Snapshot, error)
	Assigned(context.Context, string, string) (bool, error)
}

type Service struct {
	repository Repository
	location   *time.Location
	now        func() time.Time
}

func New(repository Repository, location *time.Location, now func() time.Time) *Service {
	return &Service{
		repository: repository,
		location:   location,
		now:        now,
	}
}

func (s *Service) Progress(ctx context.Context, id string) (Progress, error) {
	snapshot, err := s.repository.Load(ctx, id)
	if err != nil {
		return Progress{}, err
	}

	list := tasks.PrepareList(snapshot.Tasks, tasks.CalendarDate(s.now(), s.location))
	result := Progress{
		Intern:       snapshot.Intern,
		Mentor:       snapshot.Mentor,
		StartDate:    snapshot.Intern.StartDate,
		Goals:        snapshot.Intern.Goals,
		Expectations: snapshot.Intern.Expectations,
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
	ok, err := s.repository.Assigned(ctx, mentor, id)
	if err != nil {
		return Progress{}, err
	}

	if !ok {
		return Progress{}, apperror.ErrNotFound
	}

	return s.Progress(ctx, id)
}
