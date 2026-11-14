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
