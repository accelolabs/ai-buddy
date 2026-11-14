package progress

import (
	"ai-buddy/backend-tracker/internal/interns"
	"ai-buddy/backend-tracker/internal/tasks"
)

type Progress struct {
	Intern       interns.Profile
	Mentor       *interns.Profile
	StartDate    *string
	Goals        string
	Expectations string
	Tasks        []tasks.Task
	Completed    int
	Total        int
	Percent      int
	NextTask     *tasks.Task
}

// Snapshot is the repository data needed to calculate progress.
type Snapshot struct {
	Intern interns.Profile
	Mentor *interns.Profile
	Tasks  []tasks.Task
}
