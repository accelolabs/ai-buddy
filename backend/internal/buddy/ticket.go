package buddy

import "errors"

var ErrMentorChanged = errors.New("assigned mentor changed")

type Recipient struct {
	ID   string
	Name string
}

type Ticket struct {
	Title       string
	Description string
	Recipient   Recipient
	TicketID    *string
}
