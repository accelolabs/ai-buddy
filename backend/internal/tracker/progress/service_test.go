package progress

import (
	"context"
	"errors"
	"testing"
	"time"

	"ai-buddy/internal/tracker/apperror"
	"ai-buddy/internal/tracker/interns"
	"ai-buddy/internal/tracker/tasks"
)

type repositoryStub struct {
	snapshot struct {
		Intern interns.Profile
		Mentor *interns.Profile
		Tasks  []tasks.Task
	}
	fail         string
	err          error
	assigned     bool
	id, mentorID string
}

func (r *repositoryStub) Intern(_ context.Context, id string) (interns.Profile, error) {
	r.id = id
	if id == "j" {
		return interns.Profile{ID: id}, r.err
	}
	return r.snapshot.Intern, r.err
}
func (r *repositoryStub) Mentor(_ context.Context, id string) (*interns.Profile, error) {
	if id == "j" {
		return nil, nil
	}
	if r.fail == "mentor" {
		return nil, errors.New("mentor failure")
	}
	return r.snapshot.Mentor, nil
}
func (r *repositoryStub) Tasks(context.Context, string) ([]tasks.Task, error) {
	if r.fail == "tasks" {
		return nil, errors.New("tasks failure")
	}
	return r.snapshot.Tasks, nil
}

func (r *repositoryStub) Assigned(_ context.Context, mentor, id string) (bool, error) {
	r.mentorID = mentor
	r.id = id
	return r.assigned, r.err
}

func fixture() (*Service, *repositoryStub) {
	start := "2026-11-01"
	repo := &repositoryStub{assigned: true}
	repo.snapshot.Intern = interns.Profile{ID: "i", StartDate: &start, Goals: "Goals", Expectations: "Expectations"}
	repo.snapshot.Mentor = &interns.Profile{ID: "m"}
	loc, _ := time.LoadLocation("Europe/Moscow")
	return New(repo, repo, loc, func() time.Time {
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

func TestRepositoryErrors(t *testing.T) {
	for _, fail := range []string{"mentor", "tasks"} {
		s, repo := fixture()
		repo.fail = fail
		if _, err := s.Progress(context.Background(), "i"); err == nil {
			t.Fatal(fail)
		}
	}
}
