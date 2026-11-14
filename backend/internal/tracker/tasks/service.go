package tasks

import (
	"context"
	"strings"
	"time"

	"ai-buddy/internal/tracker/apperror"
	"ai-buddy/internal/tracker/interns"
)

type Repository interface {
	Tasks(context.Context, string) ([]Task, error)
	Create(context.Context, string, string, CreateInput) (Task, bool, error)
	Complete(context.Context, string, string) (Task, error)
}

type InternReader interface {
	Intern(context.Context, string) (interns.Profile, error)
}

type Service struct {
	repository Repository
	interns    InternReader
	location   *time.Location
	now        func() time.Time
}

func New(repository Repository, interns InternReader, location *time.Location, now func() time.Time) *Service {
	return &Service{
		repository: repository,
		interns:    interns,
		location:   location,
		now:        now,
	}
}

func (s *Service) Tasks(ctx context.Context, id string) ([]Task, error) {
	if _, err := s.interns.Intern(ctx, id); err != nil {
		return nil, err
	}

	list, err := s.repository.Tasks(ctx, id)
	if err != nil {
		return nil, err
	}

	return PrepareList(list, CalendarDate(s.now(), s.location)), nil
}

func (s *Service) Create(ctx context.Context, id, key string, in CreateInput) (Task, bool, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.Deadline = strings.TrimSpace(in.Deadline)
	if strings.TrimSpace(key) == "" || len(key) > 256 || in.Title == "" {
		return Task{}, false, apperror.ErrInvalid
	}

	if _, err := time.Parse(time.DateOnly, in.Deadline); err != nil {
		return Task{}, false, apperror.ErrInvalid
	}

	if _, err := s.interns.Intern(ctx, id); err != nil {
		return Task{}, false, err
	}

	t, created, err := s.repository.Create(ctx, id, key, in)
	return t.WithState(CalendarDate(s.now(), s.location)), created, err
}

func (s *Service) Complete(ctx context.Context, id, taskID string) (Task, error) {
	if _, err := s.interns.Intern(ctx, id); err != nil {
		return Task{}, err
	}

	t, err := s.repository.Complete(ctx, id, taskID)
	return t.WithState(CalendarDate(s.now(), s.location)), err
}
