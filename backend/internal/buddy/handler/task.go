package handler

import (
	"time"

	// Project packages.
	"backend/internal/buddy"
)

type taskResponse struct {
	Description string    `json:"description"`
	Deadline    time.Time `json:"deadline"`
}

func taskView(card *buddy.Task) *taskResponse {
	if card == nil {
		return nil
	}

	return &taskResponse{
		Description: card.Description,
		Deadline:    card.Deadline,
	}
}
