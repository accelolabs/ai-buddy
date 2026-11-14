package progress

import (
	"context"
	"errors"
	"testing"
	"time"

	"ai-buddy/backend-tracker/internal/apperror"
	"ai-buddy/backend-tracker/internal/interns"
	"ai-buddy/backend-tracker/internal/tasks"
)

type repositoryStub struct {
	snapshot     Snapshot
	err          error
	assigned     bool
	id, mentorID string
}

func (r *repositoryStub) Load(_ context.Context, id string) (Snapshot, error) {
	r.id = id
	snapshot := r.snapshot
	if id == "j" {
		snapshot.Intern = interns.Profile{ID: id}
		snapshot.Mentor = nil
	}

	return snapshot, r.err
}

func (r *repositoryStub) Assigned(_ context.Context, mentor, id string) (bool, error) {
	r.mentorID = mentor
	r.id = id
	return r.assigned, r.err
}

func fixture() (*Service, *repositoryStub) {
	start := "2026-11-01"
	repo := &repositoryStub{
		snapshot: Snapshot{
			Intern: interns.Profile{
				ID:           "i",
				StartDate:    &start,
				Goals:        "Goals",
				Expectations: "Expectations",
			},
			Mentor: &interns.Profile{ID: "m"},
		},
		assigned: true,
	}
	loc, _ := time.LoadLocation("Europe/Moscow")
	return New(repo, loc, func() time.Time {
		return time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	}), repo
}

func TestProgress(t *testing.T) {
	s, m := fixture()
	ctx := context.Background()
	p, e := s.Progress(ctx, "i")
	if e != nil ||
		p.Total != 0 ||
		p.Percent != 0 ||
		p.NextTask != nil ||
		p.Tasks == nil ||
		p.Mentor.ID != "m" {
		t.Fatalf("empty: %+v %v", p, e)
	}

	p, e = s.Progress(ctx, "j")
	if e != nil || p.Mentor != nil {
		t.Fatal("unassigned", p, e)
	}

	now := s.now()
	done := now
	m.snapshot.Tasks = []tasks.Task{
		{
			ID:        "z",
			InternID:  "i",
			Deadline:  "2026-11-13",
			CreatedAt: now.Add(time.Hour),
		},
		{
			ID:        "b",
			InternID:  "i",
			Deadline:  "2026-11-12",
			CreatedAt: now,
		},
		{
			ID:        "a",
			InternID:  "i",
			Deadline:  "2026-11-13",
			CreatedAt: now,
		},
		{
			ID:          "c",
			InternID:    "i",
			Deadline:    "2026-11-01",
			CreatedAt:   now,
			CompletedAt: &done,
		},
	}
	p, e = s.Progress(ctx, "i")
	if e != nil ||
		p.Completed != 1 ||
		p.Total != 4 ||
		p.Percent != 25 ||
		p.NextTask.ID != "a" ||
		!p.NextTask.Overdue {
		t.Fatalf("overdue order: %+v %v", p, e)
	}

	for i := range m.snapshot.Tasks {
		m.snapshot.Tasks[i].CompletedAt = nil
		m.snapshot.Tasks[i].Deadline = "2026-11-15"
	}

	m.snapshot.Tasks[0].Deadline = "2026-11-14"
	p, _ = s.Progress(ctx, "i")
	if p.NextTask.ID != "z" || p.NextTask.Overdue {
		t.Fatal("nearest", p)
	}

	m.snapshot.Tasks[0].Deadline = "2026-11-15"
	p, _ = s.Progress(ctx, "i")
	if p.NextTask.ID != "a" {
		t.Fatal("deadline ties", p)
	}

	for i := range m.snapshot.Tasks {
		m.snapshot.Tasks[i].CompletedAt = &done
	}

	p, _ = s.Progress(ctx, "i")
	if p.Percent != 100 || p.NextTask != nil {
		t.Fatal("completed", p)
	}
}

func TestMentorAccessAndErrors(t *testing.T) {
	s, repo := fixture()
	ctx := context.Background()
	p, err := s.MentorProgress(ctx, "m", "i")
	if err != nil || p.Intern.ID != "i" || repo.mentorID != "m" || repo.id != "i" {
		t.Fatal(p, err)
	}

	repo.assigned = false
	if _, err = s.MentorProgress(ctx, "m", "j"); err != apperror.ErrNotFound {
		t.Fatal(err)
	}

	repo.err = errors.New("storage failure")
	if _, err = s.MentorProgress(ctx, "m", "i"); err != repo.err {
		t.Fatal(err)
	}

	if _, err = s.Progress(ctx, "i"); err != repo.err {
		t.Fatal(err)
	}
}

func TestProgressCalendarAndProfile(t *testing.T) {
	s, repo := fixture()
	repo.snapshot.Tasks = []tasks.Task{{ID: "today", Deadline: "2026-11-14"}, {ID: "tomorrow", Deadline: "2026-11-15"}}
	calls := 0
	s.now = func() time.Time {
		calls++
		return time.Date(2026, 11, 14, 20, 59, 59, 0, time.UTC)
	}
	p, err := s.Progress(context.Background(), "i")
	if err != nil ||
		calls != 1 ||
		p.Tasks[0].Overdue ||
		p.StartDate != repo.snapshot.Intern.StartDate ||
		p.Goals != "Goals" ||
		p.Expectations != "Expectations" {
		t.Fatal(p, err, calls)
	}

	s.now = func() time.Time {
		return time.Date(2026, 11, 14, 21, 0, 0, 0, time.UTC)
	}
	p, err = s.Progress(context.Background(), "i")
	if err != nil || !p.Tasks[0].Overdue || p.Tasks[1].Overdue || p.NextTask.ID != "today" {
		t.Fatal(p, err)
	}
}
