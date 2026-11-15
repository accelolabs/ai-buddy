package ticket

import (
	"context"
	"errors"
	"strings"

	"ai-buddy/internal/users"
)

type Repository interface {
	Find(context.Context, string, string) (Ticket, error)
	Create(context.Context, string, string, string, CreateInput) (Ticket, bool, error)
	Sent(context.Context, string) ([]Ticket, error)
	Incoming(context.Context, string) ([]Ticket, error)
}

type MentorFinder interface {
	Mentor(context.Context, string) (*users.User, error)
}

type Service struct {
	repository Repository
	tracker    MentorFinder
}

func New(repository Repository, tracker MentorFinder) *Service {
	return &Service{
		repository: repository,
		tracker:    tracker,
	}
}

func (s *Service) Create(ctx context.Context, sender, key string, input CreateInput) (Ticket, bool, error) {
	input.Question = strings.TrimSpace(input.Question)
	input.Description = strings.TrimSpace(input.Description)
	if strings.TrimSpace(sender) == "" || strings.TrimSpace(key) == "" || len(key) > 256 || input.Question == "" || input.Description == "" {
		return Ticket{}, false, ErrInvalid
	}

	saved, err := s.repository.Find(ctx, sender, key)
	if err == nil {
		if saved.Question != input.Question || saved.Description != input.Description {
			return Ticket{}, false, ErrConflict
		}
		return saved, false, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Ticket{}, false, err
	}

	recipient, err := s.tracker.Mentor(ctx, sender)
	if err != nil {
		return Ticket{}, false, err
	}
	if recipient == nil {
		return Ticket{}, false, ErrNoMentor
	}

	return s.repository.Create(ctx, sender, recipient.ID, key, input)
}

func (s *Service) Sent(ctx context.Context, id string) ([]Ticket, error) {
	return s.repository.Sent(ctx, id)
}

func (s *Service) Incoming(ctx context.Context, id string) ([]Ticket, error) {
	return s.repository.Incoming(ctx, id)
}
