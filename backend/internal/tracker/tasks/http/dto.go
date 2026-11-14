package taskshttp

import (
	"ai-buddy/internal/tracker/dto"
)

type tasksResponse struct {
	Tasks []dto.Task `json:"tasks"`
}
