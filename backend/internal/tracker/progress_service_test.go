package tracker

import (
	"ai-buddy/internal/users"
	"context"
	"errors"
	"testing"
	"time"
)

type repositoryStub struct {
	snapshot struct {
		Intern users.User
		Mentor *users.User
		Tasks  []Task
	}
	fail         string
	err          error
	assigned     bool
	id, mentorID string
}

func (r *repositoryStub) Get(_ context.Context, id string) (users.User, error) {
	r.id = id
	if id == "j" {
		return users.User{ID: id, Role: users.RoleIntern}, r.err
	}

	return r.snapshot.Intern, r.err
}

//nolint:nilnil // This fixture models an unassigned mentor.
func (r *repositoryStub) Mentor(_ context.Context, id string) (*users.User, error) {
	if id == "j" {
		return nil, nil
	}

	if r.fail == "mentor" {
		return nil, errors.New("mentor failure")
	}

	return r.snapshot.Mentor, nil
}

func (r *repositoryStub) Tasks(context.Context, string) ([]Task, error) {
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

func fixture(t *testing.T) (*Service, *repositoryStub) {
	start := "2026-11-01"
	repo := &repositoryStub{assigned: true}
	repo.snapshot.Intern = users.User{
		ID:           "i",
		Role:         users.RoleIntern,
		StartDate:    &start,
		Goals:        "Goals",
		Expectations: "Expectations",
	}
	repo.snapshot.Mentor = &users.User{ID: "m"}

	loc, locationErr := time.LoadLocation("Europe/Moscow")
	if locationErr != nil {
		t.Fatal(locationErr)
	}

	return newTestService(repo, repo, loc, func() time.Time {
		return time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	}), repo
}

func TestMentorAccessAndErrors(t *testing.T) {
	s, repo := fixture(t)
	ctx := context.Background()

	p, err := s.MentorProgress(ctx, "m", "i")
	if err != nil || p.Intern.ID != "i" || repo.mentorID != "m" || repo.id != "i" {
		t.Fatal(p, err)
	}

	repo.assigned = false

	if _, err = s.MentorProgress(ctx, "m", "j"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}

	repo.err = errors.New("storage failure")
	if _, err = s.MentorProgress(ctx, "m", "i"); !errors.Is(err, repo.err) {
		t.Fatal(err)
	}

	if _, err = s.Progress(ctx, "i"); !errors.Is(err, repo.err) {
		t.Fatal(err)
	}
}

func newTestService(
	store Repository,
	profiles Profiles,
	loc *time.Location,
	now func() time.Time,
) *Service {
	return &Service{Store: store, Profiles: profiles, Location: loc, Now: now}
}

func (*repositoryStub) Create(context.Context, string, string, CreateInput) (Task, bool, error) {
	panic("unused")
}
func (*repositoryStub) Complete(context.Context, string, string) (Task, error) { panic("unused") }
func (*repositoryStub) Interns(context.Context, string) ([]users.User, error)  { panic("unused") }
