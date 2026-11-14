package taskshttp

import (
	"time"

	"ai-buddy/internal/tracker/tasks"
)

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

type tasksResponse struct {
	Tasks []taskResponse `json:"tasks"`
}
