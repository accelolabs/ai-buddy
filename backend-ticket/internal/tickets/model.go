package tickets

import "time"

type Ticket struct {
	ID          string
	SenderID    string
	RecipientID string
	Question    string
	Description string
	CreatedAt   time.Time
}

type CreateInput struct {
	Question    string
	Description string
}
