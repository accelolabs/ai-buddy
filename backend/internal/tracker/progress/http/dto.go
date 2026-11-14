package progresshttp

import (
	"ai-buddy/internal/tracker/dto"

	"ai-buddy/internal/tracker/progress"
)

type progressResponse struct {
	Intern       dto.Profile  `json:"intern"`
	Mentor       *dto.Profile `json:"mentor"`
	StartDate    *string      `json:"start_date"`
	Goals        string       `json:"goals"`
	Expectations string       `json:"expectations"`
	Tasks        []dto.Task   `json:"tasks"`
	Completed    int          `json:"completed"`
	Total        int          `json:"total"`
	Percent      int          `json:"percent"`
	NextTask     *dto.Task    `json:"next_task"`
}

func fromProgress(model progress.Progress) progressResponse {
	var mentor *dto.Profile
	if model.Mentor != nil {
		converted := dto.FromProfile(*model.Mentor)
		mentor = &converted
	}

	var nextTask *dto.Task
	if model.NextTask != nil {
		converted := dto.FromTask(*model.NextTask)
		nextTask = &converted
	}

	return progressResponse{
		Intern:       dto.FromProfile(model.Intern),
		Mentor:       mentor,
		StartDate:    model.StartDate,
		Goals:        model.Goals,
		Expectations: model.Expectations,
		Tasks:        dto.Tasks(model.Tasks),
		Completed:    model.Completed,
		Total:        model.Total,
		Percent:      model.Percent,
		NextTask:     nextTask,
	}
}
