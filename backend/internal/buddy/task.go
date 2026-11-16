package buddy

import "time"

type Task struct {
	Description string
	Deadline    time.Time
}
