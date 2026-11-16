package tracker

import "time"

type Task struct {
	ID          string
	UserID      string
	Description string
	Deadline    time.Time
	Status      string
	CreatedAt   time.Time
}
