package tracker

import (
	"backend-ai/internal/users"
)

type Progress struct {
	Intern       users.User  `json:"intern"`
	Mentor       *users.User `json:"mentor"`
	StartDate    *string     `json:"start_date"`
	Goals        string      `json:"goals"`
	Expectations string      `json:"expectations"`
	Tasks        []Task      `json:"tasks"`
	Completed    int         `json:"completed"`
	Total        int         `json:"total"`
	Percent      int         `json:"percent"`
	NextTask     *Task       `json:"next_task"`
}
