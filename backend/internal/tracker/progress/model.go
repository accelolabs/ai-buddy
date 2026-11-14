package progress

import (
	"ai-buddy/internal/tracker/interns"
	"ai-buddy/internal/tracker/tasks"
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
