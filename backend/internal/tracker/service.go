package tracker

import (
	"ai-buddy/internal/users"
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type Repository interface {
	// Tasks returns a fresh caller-owned slice ordered by created_at, then ID.
	Tasks(context.Context, string) ([]Task, error)
	Create(context.Context, string, string, CreateInput) (Task, bool, error)
	Complete(context.Context, string, string) (Task, error)
}

type Profiles interface {
	Get(context.Context, string) (users.User, error)
	Mentor(context.Context, string) (*users.User, error)
	Assigned(context.Context, string, string) (bool, error)
	Interns(context.Context, string) ([]users.User, error)
}
type Service struct {
	Store    Repository
	Profiles Profiles
	Location *time.Location
	Now      func() time.Time
}

func (s *Service) CreateTask(
	ctx context.Context,
	id, key string,
	in CreateInput,
) (Task, bool, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)

	in.Deadline = strings.TrimSpace(in.Deadline)
	if strings.TrimSpace(key) == "" || len(key) > 256 || in.Title == "" {
		return Task{}, false, ErrInvalid
	}

	if _, err := time.Parse(time.DateOnly, in.Deadline); err != nil {
		return Task{}, false, ErrInvalid
	}

	t, created, err := s.Store.Create(ctx, id, key, in)

	return t.WithState(CalendarDate(s.Now(), s.Location)), created, err
}

func (s *Service) CompleteTask(ctx context.Context, id, taskID string) (Task, error) {
	t, err := s.Store.Complete(ctx, id, taskID)
	return t.WithState(CalendarDate(s.Now(), s.Location)), err
}

func (s *Service) Progress(ctx context.Context, id string) (Progress, error) {
	intern, err := s.Profiles.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && intern.Role != users.RoleIntern) {
		return Progress{}, ErrNotFound
	}

	if err != nil {
		return Progress{}, err
	}

	mentor, err := s.Profiles.Mentor(ctx, id)
	if err != nil {
		return Progress{}, err
	}

	list, err := s.Store.Tasks(ctx, id)
	if err != nil {
		return Progress{}, err
	}

	if list == nil {
		list = []Task{}
	}

	today := CalendarDate(s.Now(), s.Location)

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
		list[i] = list[i].WithState(today)

		task := &list[i]
		if task.CompletedAt != nil {
			result.Completed++
			continue
		}

		next := result.NextTask
		if next == nil ||
			(!next.Overdue && (task.Overdue || task.Deadline < next.Deadline)) {
			result.NextTask = task
		}
	}

	if result.Total > 0 {
		result.Percent = 100 * result.Completed / result.Total
	}

	return result, nil
}

func (s *Service) MentorProgress(ctx context.Context, mentor, id string) (Progress, error) {
	ok, err := s.Profiles.Assigned(ctx, mentor, id)
	if err != nil {
		return Progress{}, err
	}

	if !ok {
		return Progress{}, ErrNotFound
	}

	return s.Progress(ctx, id)
}
