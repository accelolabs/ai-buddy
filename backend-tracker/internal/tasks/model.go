package tasks

import (
	"sort"
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
	Title       string
	Description string
	Deadline    string
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

// PrepareList applies the common list order and state calculation.
func PrepareList(list []Task, today string) []Task {
	// Keep the repository snapshot unchanged while producing a non-null JSON list.
	list = append([]Task{}, list...)
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].CreatedAt.Equal(list[j].CreatedAt) {
			return list[i].ID < list[j].ID
		}

		return list[i].CreatedAt.Before(list[j].CreatedAt)
	})
	for i := range list {
		list[i] = list[i].WithState(today)
	}

	return list
}

// CalendarDate keeps timezone handling identical for tasks and progress.
func CalendarDate(now time.Time, location *time.Location) string {
	return now.In(location).Format(time.DateOnly)
}
