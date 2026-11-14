package postgres

import (
	"context"
	"errors"
	"testing"

	"ai-buddy/backend-tracker/internal/interns"
	"ai-buddy/backend-tracker/internal/tasks"
)

type internRepositoryStub struct {
	fail         string
	err          error
	calls        []string
	mentor       *interns.Profile
	mentorID, id string
}

func (r *internRepositoryStub) Intern(_ context.Context, id string) (interns.Profile, error) {
	r.calls = append(r.calls, "intern")
	r.id = id
	if r.fail == "intern" {
		return interns.Profile{}, r.err
	}

	return interns.Profile{ID: id}, nil
}

func (r *internRepositoryStub) Mentor(_ context.Context, id string) (*interns.Profile, error) {
	r.calls = append(r.calls, "mentor")
	r.id = id
	if r.fail == "mentor" {
		return nil, r.err
	}

	return r.mentor, nil
}

func (r *internRepositoryStub) Assigned(_ context.Context, mentor, id string) (bool, error) {
	r.mentorID = mentor
	r.id = id
	return mentor == "m" && id == "i", r.err
}

type taskRepositoryStub struct {
	err    error
	called bool
	id     string
}

func (r *taskRepositoryStub) Tasks(_ context.Context, id string) ([]tasks.Task, error) {
	r.called = true
	r.id = id
	return []tasks.Task{{ID: "task"}}, r.err
}

func TestComposition(t *testing.T) {
	for _, fail := range []string{
		"",
		"intern",
		"mentor",
		"tasks",
	} {
		t.Run(fail, func(t *testing.T) {
			want := errors.New("storage failure")
			internRepo := &internRepositoryStub{
				fail:   fail,
				err:    want,
				mentor: &interns.Profile{ID: "m"},
			}
			taskRepo := &taskRepositoryStub{}
			if fail == "tasks" {
				taskRepo.err = want
			}

			repo := New(internRepo, taskRepo)
			snapshot, err := repo.Load(context.Background(), "i")
			if fail != "" {
				if err != want {
					t.Fatal(err)
				}
			} else if err != nil ||
				snapshot.Intern.ID != "i" ||
				snapshot.Mentor.ID != "m" ||
				len(snapshot.Tasks) != 1 ||
				taskRepo.id != "i" {
				t.Fatal(snapshot, err)
			}

			if (fail == "intern" &&
				len(internRepo.calls) != 1) ||
				((fail == "intern" ||
					fail == "mentor") &&
					taskRepo.called) {
				t.Fatal("read continued after error")
			}
		})
	}

	repo := New(&internRepositoryStub{}, &taskRepositoryStub{})
	snapshot, err := repo.Load(context.Background(), "i")
	if err != nil || snapshot.Mentor != nil {
		t.Fatal(snapshot, err)
	}
}

func TestAssigned(t *testing.T) {
	internRepo := &internRepositoryStub{}
	repo := New(internRepo, &taskRepositoryStub{})
	ctx := context.Background()
	ok, err := repo.Assigned(ctx, "m", "i")
	if err != nil || !ok || internRepo.mentorID != "m" || internRepo.id != "i" {
		t.Fatal(ok, err)
	}

	ok, err = repo.Assigned(ctx, "m", "other")
	if err != nil || ok {
		t.Fatal(ok, err)
	}

	want := errors.New("storage error")
	internRepo.err = want
	if _, err = repo.Assigned(ctx, "m", "i"); err != want {
		t.Fatal(err)
	}
}
