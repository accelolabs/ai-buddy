package ticketshttp

import "ai-buddy/internal/ticket/tickets"

type ticketsResponse struct {
	Tickets []tickets.Ticket `json:"tickets"`
}
