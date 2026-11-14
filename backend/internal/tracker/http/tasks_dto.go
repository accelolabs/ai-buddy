package trackerhttp

import (
	"ai-buddy/internal/tracker/tasks"
)

type tasksResponse struct {
	Tasks []tasks.Task `json:"tasks"`
}
