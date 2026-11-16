package ticket

import "time"

type Ticket struct {
	ID          string
	UserID      string
	Title       string
	Description string
	CreatedAt   time.Time
}
