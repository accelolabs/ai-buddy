package progresshttp

import (
	"time"

	"ai-buddy/internal/tracker/interns"
	"ai-buddy/internal/tracker/progress"
	"ai-buddy/internal/tracker/tasks"
)

type progressResponse struct {
	Intern       profileResponse  `json:"intern"`
	Mentor       *profileResponse `json:"mentor"`
	StartDate    *string          `json:"start_date"`
	Goals        string           `json:"goals"`
	Expectations string           `json:"expectations"`
	Tasks        []taskResponse   `json:"tasks"`
	Completed    int              `json:"completed"`
	Total        int              `json:"total"`
	Percent      int              `json:"percent"`
	NextTask     *taskResponse    `json:"next_task"`
}

func fromProgress(model progress.Progress) progressResponse {
	var mentor *profileResponse
	if model.Mentor != nil {
		converted := fromProfile(*model.Mentor)
		mentor = &converted
	}

	var nextTask *taskResponse
	if model.NextTask != nil {
		converted := fromTask(*model.NextTask)
		nextTask = &converted
	}

	return progressResponse{
		Intern:       fromProfile(model.Intern),
		Mentor:       mentor,
		StartDate:    model.StartDate,
		Goals:        model.Goals,
		Expectations: model.Expectations,
		Tasks:        fromTasks(model.Tasks),
		Completed:    model.Completed,
		Total:        model.Total,
		Percent:      model.Percent,
		NextTask:     nextTask,
	}
}

type taskResponse struct {
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

func fromTask(model tasks.Task) taskResponse {
	return taskResponse{
		ID:          model.ID,
		InternID:    model.InternID,
		Title:       model.Title,
		Description: model.Description,
		Deadline:    model.Deadline,
		CreatedAt:   model.CreatedAt,
		CompletedAt: model.CompletedAt,
		Status:      model.Status,
		Overdue:     model.Overdue,
	}
}

func fromTasks(models []tasks.Task) []taskResponse {
	if models == nil {
		return nil
	}

	result := make([]taskResponse, len(models))
	for i, model := range models {
		result[i] = fromTask(model)
	}

	return result
}

type profileResponse struct {
	ID           string  `json:"authentik_id"`
	Name         string  `json:"name"`
	Username     string  `json:"username"`
	Email        string  `json:"email"`
	Role         string  `json:"role"`
	Team         string  `json:"team"`
	StartDate    *string `json:"start_date"`
	Goals        string  `json:"goals"`
	Expectations string  `json:"expectations"`
}

func fromProfile(model interns.Profile) profileResponse {
	return profileResponse{
		ID:           model.ID,
		Name:         model.Name,
		Username:     model.Username,
		Email:        model.Email,
		Role:         model.Role,
		Team:         model.Team,
		StartDate:    model.StartDate,
		Goals:        model.Goals,
		Expectations: model.Expectations,
	}
}
