package progress

import (
	"ai-buddy/internal/tracker/interns"
	"ai-buddy/internal/tracker/tasks"
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
