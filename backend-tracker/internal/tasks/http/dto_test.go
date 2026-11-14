package taskshttp

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"ai-buddy/backend-tracker/internal/tasks"
)

func checkJSON(t *testing.T, response any, want string) {
	t.Helper()
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(raw, &gotValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("got %s\nwant %s", raw, want)
	}
}

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
	checkJSON(t, fromTask(task), taskJSON)
	checkJSON(t, tasksResponse{Tasks: fromTasks([]tasks.Task{task})}, `{"tasks":[`+taskJSON+`]}`)
	checkJSON(t, tasksResponse{Tasks: fromTasks(nil)}, `{"tasks":null}`)
	checkJSON(t, tasksResponse{Tasks: fromTasks([]tasks.Task{})}, `{"tasks":[]}`)
	task.CompletedAt = nil
	checkJSON(t, fromTask(task), `{"id":"task","intern_id":"intern","title":"Question","description":"Details","deadline":"2026-11-15","created_at":"2026-11-14T12:30:00+03:00","completed_at":null,"status":"completed","overdue":false}`)
}
