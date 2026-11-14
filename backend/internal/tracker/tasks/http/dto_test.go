package taskshttp

import (
	"testing"
	"time"

	"ai-buddy/internal/testutil"
	"ai-buddy/internal/tracker/tasks"
)

func TestResponseContracts(t *testing.T) {
	created := time.Date(2026, 11, 14, 12, 30, 0, 0, time.FixedZone("MSK", 3*60*60))
	completed := created.Add(time.Hour)
	task := tasks.Task{
		ID:          "task",
		InternID:    "intern",
		Title:       "Question",
		Description: "Details",
		Deadline:    "2026-11-15",
		CreatedAt:   created,
		CompletedAt: &completed,
		Status:      "completed",
		Overdue:     false,
	}
	const taskJSON = `{"id":"task","intern_id":"intern","title":"Question","description":"Details","deadline":"2026-11-15","created_at":"2026-11-14T12:30:00+03:00","completed_at":"2026-11-14T13:30:00+03:00","status":"completed","overdue":false}`
	testutil.JSON(t, fromTask(task), taskJSON)
	testutil.JSON(t, tasksResponse{Tasks: fromTasks([]tasks.Task{task})}, `{"tasks":[`+taskJSON+`]}`)
	testutil.JSON(t, tasksResponse{Tasks: fromTasks(nil)}, `{"tasks":null}`)
	testutil.JSON(t, tasksResponse{Tasks: fromTasks([]tasks.Task{})}, `{"tasks":[]}`)
	task.CompletedAt = nil
	testutil.JSON(t, fromTask(task), `{"id":"task","intern_id":"intern","title":"Question","description":"Details","deadline":"2026-11-15","created_at":"2026-11-14T12:30:00+03:00","completed_at":null,"status":"completed","overdue":false}`)
}
