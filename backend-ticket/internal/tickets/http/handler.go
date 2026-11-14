package ticketshttp

import (
	"context"

	"ai-buddy/backend-ticket/internal/tickets"
)

type Service interface {
	Create(context.Context, string, string, tickets.CreateInput) (tickets.Ticket, bool, error)
	Sent(context.Context, string) ([]tickets.Ticket, error)
	Incoming(context.Context, string) ([]tickets.Ticket, error)
}

type Handler struct{ service Service }

func New(service Service) *Handler { return &Handler{service: service} }
