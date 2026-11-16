package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"strings"

	// Project packages.
	"backend/internal/ticket"
	"backend/internal/users"
)

var (
	ErrInvalidInput = errors.New("invalid ticket")
	ErrForbidden    = errors.New("only interns can create tickets")
	ErrNotFound     = errors.New("ticket not found")
)

type ticketRepository interface {
	Add(context.Context, ticket.Ticket) (ticket.Ticket, error)
	List(context.Context, users.User) ([]ticket.Ticket, error)
	Get(context.Context, users.User, string) (ticket.Ticket, error)
}

type Service struct {
	repository ticketRepository
}

func NewService(repository ticketRepository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(ctx context.Context, user users.User, title, description string) (ticket.Ticket, error) {
	if user.Role != users.RoleIntern {
		return ticket.Ticket{}, ErrForbidden
	}

	title, description = strings.TrimSpace(title), strings.TrimSpace(description)
	if title == "" || description == "" {
		return ticket.Ticket{}, ErrInvalidInput
	}

	return s.repository.Add(ctx, ticket.Ticket{
		ID:          rand.Text(),
		UserID:      user.ID,
		Title:       title,
		Description: description,
	})
}

func (s *Service) List(ctx context.Context, user users.User) ([]ticket.Ticket, error) {
	return s.repository.List(ctx, user)
}

func (s *Service) Get(ctx context.Context, user users.User, id string) (ticket.Ticket, error) {
	value, err := s.repository.Get(ctx, user, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ticket.Ticket{}, ErrNotFound
	}

	return value, err
}
