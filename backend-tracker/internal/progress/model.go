package progress

import (
	"ai-buddy/backend-tracker/internal/interns"
	"ai-buddy/backend-tracker/internal/tasks"
)

type Progress struct {
	Intern       interns.Profile  `json:"intern"`
	Mentor       *interns.Profile `json:"mentor"`
	StartDate    *string          `json:"start_date"`
	Goals        string           `json:"goals"`
	Expectations string           `json:"expectations"`
	Tasks        []tasks.Task     `json:"tasks"`
	Completed    int              `json:"completed"`
	Total        int              `json:"total"`
	Percent      int              `json:"percent"`
	NextTask     *tasks.Task      `json:"next_task"`
}

// Snapshot is the repository data needed to calculate progress.
type Snapshot struct {
	Intern interns.Profile
	Mentor *interns.Profile
	Tasks  []tasks.Task
}
