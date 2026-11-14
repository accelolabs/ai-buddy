package interns

import (
	"context"
	"errors"
	"testing"

	"ai-buddy/backend-tracker/internal/apperror"
)

type repositoryStub struct {
	internErr, mentorErr, listErr error
	mentor                        *Profile
	id                            string
	mentorCalled                  bool
}

func (r *repositoryStub) Intern(_ context.Context, id string) (Profile, error) {
	r.id = id
	return Profile{ID: id, Role: "intern"}, r.internErr
}
func (r *repositoryStub) Mentor(_ context.Context, id string) (*Profile, error) {
	r.id = id
	r.mentorCalled = true
	return r.mentor, r.mentorErr
}
func (r *repositoryStub) Interns(_ context.Context, id string) ([]Profile, error) {
	r.id = id
	return []Profile{{ID: "i"}}, r.listErr
}
func TestService(t *testing.T) {
	ctx := context.Background()
	repo := &repositoryStub{mentor: &Profile{ID: "m"}}
	s := New(repo)
	intern, err := s.Intern(ctx, "i")
	if err != nil || intern.ID != "i" || repo.id != "i" {
		t.Fatal(intern, err)
	}
	mentor, err := s.Mentor(ctx, "i")
	if err != nil || mentor.ID != "m" || repo.id != "i" {
		t.Fatal(mentor, err)
	}
	repo.mentor = nil
	mentor, err = s.Mentor(ctx, "j")
	if err != nil || mentor != nil {
		t.Fatal(mentor, err)
	}
	list, err := s.Interns(ctx, "m")
	if err != nil || len(list) != 1 || repo.id != "m" {
		t.Fatal(list, err)
	}
	for _, want := range []error{apperror.ErrNotFound, errors.New("db error")} {
		repo.internErr = want
		repo.mentorCalled = false
		if _, err = s.Intern(ctx, "missing"); err != want {
			t.Fatal(err)
		}
		if _, err = s.Mentor(ctx, "missing"); err != want || repo.mentorCalled {
			t.Fatal("profile check", err)
		}
		repo.internErr = nil
		repo.mentorErr = want
		repo.listErr = want
		if _, err = s.Mentor(ctx, "i"); err != want {
			t.Fatal(err)
		}
		if _, err = s.Interns(ctx, "m"); err != want {
			t.Fatal(err)
		}
	}
}
