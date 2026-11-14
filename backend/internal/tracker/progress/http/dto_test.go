package progresshttp

import (
	"testing"
	"time"

	"ai-buddy/internal/testutil"
	"ai-buddy/internal/tracker/interns"
	"ai-buddy/internal/tracker/progress"
	"ai-buddy/internal/tracker/tasks"
)

func TestResponseContracts(t *testing.T) {
	created := time.Date(2026, 11, 14, 12, 30, 0, 0, time.FixedZone("MSK", 3*60*60))
	completed := created.Add(time.Hour)
	start := "2026-11-01"
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
	profile := interns.Profile{
		ID:           "intern",
		Name:         "Name",
		Username:     "username",
		Email:        "user@example.com",
		Role:         "intern",
		Team:         "backend",
		StartDate:    &start,
		Goals:        "goals",
		Expectations: "expectations",
	}
	const taskJSON = `{"id":"task","intern_id":"intern","title":"Question","description":"Details","deadline":"2026-11-15","created_at":"2026-11-14T12:30:00+03:00","completed_at":"2026-11-14T13:30:00+03:00","status":"completed","overdue":false}`
	const profileJSON = `{"authentik_id":"intern","name":"Name","username":"username","email":"user@example.com","role":"intern","team":"backend","start_date":"2026-11-01","goals":"goals","expectations":"expectations"}`
	snapshot := progress.Progress{
		Intern:       profile,
		Mentor:       &profile,
		StartDate:    &start,
		Goals:        "goals",
		Expectations: "expectations",
		Tasks:        []tasks.Task{task},
		Completed:    1,
		Total:        2,
		Percent:      50,
		NextTask:     &task,
	}
	testutil.JSON(t, fromProgress(snapshot), `{"intern":`+profileJSON+`,"mentor":`+profileJSON+`,"start_date":"2026-11-01","goals":"goals","expectations":"expectations","tasks":[`+taskJSON+`],"completed":1,"total":2,"percent":50,"next_task":`+taskJSON+`}`)
	snapshot.Mentor = nil
	snapshot.NextTask = nil
	snapshot.StartDate = nil
	snapshot.Tasks = []tasks.Task{}
	testutil.JSON(t, fromProgress(snapshot), `{"intern":`+profileJSON+`,"mentor":null,"start_date":null,"goals":"goals","expectations":"expectations","tasks":[],"completed":1,"total":2,"percent":50,"next_task":null}`)
	task.CompletedAt = nil
	testutil.JSON(t, fromTask(task), `{"id":"task","intern_id":"intern","title":"Question","description":"Details","deadline":"2026-11-15","created_at":"2026-11-14T12:30:00+03:00","completed_at":null,"status":"completed","overdue":false}`)
	profile.StartDate = nil
	testutil.JSON(t, fromProfile(profile), `{"authentik_id":"intern","name":"Name","username":"username","email":"user@example.com","role":"intern","team":"backend","start_date":null,"goals":"goals","expectations":"expectations"}`)
	snapshot.Tasks = nil
	testutil.JSON(t, fromProgress(snapshot), `{"intern":`+profileJSON+`,"mentor":null,"start_date":null,"goals":"goals","expectations":"expectations","tasks":null,"completed":1,"total":2,"percent":50,"next_task":null}`)
}
