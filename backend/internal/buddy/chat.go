package buddy

import "time"

type Chat struct {
	ID        string
	UserID    string
	Title     string
	CreatedAt time.Time
}
