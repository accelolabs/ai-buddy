package service

import (
	"context"
	"database/sql"
	"errors"

	// Project packages.
	"backend/internal/users"
)

var (
	ErrForbidden = errors.New("only mentors can list interns")
	ErrNotFound  = errors.New("user not found")
)

type userRepository interface {
	List(context.Context) ([]users.User, error)
	ListInterns(context.Context, string) ([]users.User, error)
	Get(context.Context, string) (users.User, error)
	GetMentor(context.Context, string) (users.User, error)
}

type Service struct {
	repository userRepository
}

func NewService(repository userRepository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) List(ctx context.Context) ([]users.User, error) {
	return s.repository.List(ctx)
}

func (s *Service) ListInterns(ctx context.Context, user users.User) ([]users.User, error) {
	if user.Role != users.RoleMentor {
		return nil, ErrForbidden
	}

	return s.repository.ListInterns(ctx, user.ID)
}

func (s *Service) Get(ctx context.Context, id string) (users.User, error) {
	value, err := s.repository.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return users.User{}, ErrNotFound
	}

	return value, err
}

func (s *Service) GetMentor(ctx context.Context, internID string) (users.User, error) {
	value, err := s.repository.GetMentor(ctx, internID)
	if errors.Is(err, sql.ErrNoRows) {
		return users.User{}, ErrNotFound
	}

	return value, err
}
