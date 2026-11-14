package progress

import (
	"context"
	"testing"
	"time"

	"ai-buddy/internal/tracker/tasks"
)

func TestNextTaskOrderAndNulls(t *testing.T) {
	s, repo := fixture()
	done := time.Now()
	for _, tc := range []struct {
		list               []tasks.Task
		next               string
		completed, percent int
	}{
		{nil, "", 0, 0},
		{[]tasks.Task{{ID: "done", CompletedAt: &done, Deadline: "2020-01-01"}}, "", 1, 100},
		{[]tasks.Task{{ID: "first", Deadline: "2026-11-13"}, {ID: "older-deadline", Deadline: "2026-11-12"}}, "first", 0, 0},
		{[]tasks.Task{{ID: "later", Deadline: "2026-11-20"}, {ID: "nearest", Deadline: "2026-11-16"}, {ID: "tie", Deadline: "2026-11-16"}}, "nearest", 0, 0},
	} {
		repo.snapshot.Tasks = tc.list
		repo.snapshot.Mentor = nil
		repo.snapshot.Intern.StartDate = nil
		p, err := s.Progress(context.Background(), "i")
		if err != nil || p.Tasks == nil || p.Mentor != nil || p.StartDate != nil || p.Completed != tc.completed || p.Percent != tc.percent {
			t.Fatal(p, err)
		}
		next := ""
		if p.NextTask != nil {
			next = p.NextTask.ID
		}
		if next != tc.next {
			t.Fatal(next, tc.next)
		}
	}
}
