package tracker

import (
	"time"
)

type Task struct {
	ID          string     `json:"id"`
	InternID    string     `json:"intern_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Deadline    string     `json:"deadline"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Status      string     `json:"status"`
	Overdue     bool       `json:"overdue"`
}

type CreateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Deadline    string `json:"deadline"`
}

// WithState derives task state from a calendar date in the application timezone.
func (t Task) WithState(today string) Task {
	t.Status = "pending"
	if t.CompletedAt != nil {
		t.Status = "completed"
	}

	t.Overdue = t.CompletedAt == nil && t.Deadline < today
	return t
}

// CalendarDate keeps timezone handling identical for tasks and progress.
func CalendarDate(now time.Time, location *time.Location) string {
	return now.In(location).Format(time.DateOnly)
}
