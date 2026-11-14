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
	return &Service{repository: repository, location: location, now: now}
}
func (s *Service) Progress(ctx context.Context, id string) (Progress, error) {
	snapshot, err := s.repository.Load(ctx, id)
	if err != nil {
		return Progress{}, err
	}
	ts := tasks.PrepareList(snapshot.Tasks, tasks.CalendarDate(s.now(), s.location))
	out := Progress{Intern: snapshot.Intern, Mentor: snapshot.Mentor, StartDate: snapshot.Intern.StartDate, Goals: snapshot.Intern.Goals, Expectations: snapshot.Intern.Expectations, Tasks: ts, Total: len(ts)}
	for i := range ts {
		t := &ts[i]
		if t.CompletedAt != nil {
			out.Completed++
			continue
		}
		if out.NextTask == nil || (!out.NextTask.Overdue && (t.Overdue || t.Deadline < out.NextTask.Deadline)) {
			out.NextTask = t
		}
	}
	if out.Total > 0 {
		out.Percent = 100 * out.Completed / out.Total
	}
	return out, nil
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
