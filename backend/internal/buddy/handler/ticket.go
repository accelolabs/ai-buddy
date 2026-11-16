package handler

import "backend/internal/buddy"

type recipientResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ticketResponse struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Recipient   recipientResponse `json:"recipient"`
}

func ticketView(card *buddy.Ticket) *ticketResponse {
	if card == nil {
		return nil
	}

	return &ticketResponse{
		Title:       card.Title,
		Description: card.Description,
		Recipient:   recipientResponse(card.Recipient),
	}
}
