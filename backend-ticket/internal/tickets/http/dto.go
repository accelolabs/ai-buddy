package ticketshttp

import (
	"time"

	"ai-buddy/backend-ticket/internal/tickets"
)

type ticketResponse struct {
	ID          string    `json:"id"`
	SenderID    string    `json:"sender_id"`
	RecipientID string    `json:"recipient_id"`
	Question    string    `json:"question"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type ticketsResponse struct {
	Tickets []ticketResponse `json:"tickets"`
}

func fromTicket(model tickets.Ticket) ticketResponse {
	return ticketResponse{
		ID:          model.ID,
		SenderID:    model.SenderID,
		RecipientID: model.RecipientID,
		Question:    model.Question,
		Description: model.Description,
		CreatedAt:   model.CreatedAt,
	}
}

func fromTickets(models []tickets.Ticket) ticketsResponse {
	result := make([]ticketResponse, len(models))
	for i, model := range models {
		result[i] = fromTicket(model)
	}

	return ticketsResponse{Tickets: result}
}
