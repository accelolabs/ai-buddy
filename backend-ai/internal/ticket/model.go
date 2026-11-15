// Package ticket handles questions sent to assigned mentors.
package ticket

import (
	"time"
)

type Ticket struct {
	ID          string    `json:"id"`
	SenderID    string    `json:"sender_id"`
	RecipientID string    `json:"recipient_id"`
	Question    string    `json:"question"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateInput struct {
	Question    string `json:"question"`
	Description string `json:"description"`
}
