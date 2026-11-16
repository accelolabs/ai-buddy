package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"strings"
	"time"

	// Project packages.
	"backend/internal/tracker"
	"backend/internal/users"
)

var (
	ErrInvalidInput = errors.New("invalid task")
	ErrForbidden    = errors.New("role is not allowed")
	ErrNotFound     = errors.New("task not found")
)

type taskRepository interface {
	Add(context.Context, tracker.Task) (tracker.Task, error)
	List(context.Context, users.User, string) ([]tracker.Task, error)
	Complete(context.Context, users.User, string) (tracker.Task, error)
}

type Service struct {
	repository taskRepository
}

func NewService(repository taskRepository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(ctx context.Context, user users.User, description, deadline string) (tracker.Task, error) {
	if user.Role != users.RoleIntern {
		return tracker.Task{}, ErrForbidden
	}

	description = strings.TrimSpace(description)
	parsed, err := time.Parse(time.RFC3339, deadline)
	if err != nil || description == "" {
		return tracker.Task{}, ErrInvalidInput
	}

	return s.repository.Add(ctx, tracker.Task{
		ID:          rand.Text(),
		UserID:      user.ID,
		Description: description,
		Deadline:    parsed,
	})
}

func (s *Service) List(ctx context.Context, user users.User) ([]tracker.Task, error) {
	if user.Role != users.RoleIntern {
		return nil, ErrForbidden
	}

	return s.list(ctx, user, user.ID)
}

func (s *Service) ListIntern(ctx context.Context, user users.User, internID string) ([]tracker.Task, error) {
	if user.Role != users.RoleMentor {
		return nil, ErrForbidden
	}

	return s.list(ctx, user, internID)
}

func (s *Service) list(ctx context.Context, user users.User, internID string) ([]tracker.Task, error) {
	values, err := s.repository.List(ctx, user, internID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}

	return values, err
}

func (s *Service) Complete(ctx context.Context, user users.User, id string) (tracker.Task, error) {
	if user.Role != users.RoleIntern {
		return tracker.Task{}, ErrForbidden
	}

	value, err := s.repository.Complete(ctx, user, id)
	if errors.Is(err, sql.ErrNoRows) {
		return tracker.Task{}, ErrNotFound
	}

	return value, err
}
