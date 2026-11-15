package tracker

import (
	"backend-ai/internal/users"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type taskRepositoryStub struct {
	input        CreateInput
	calls        int
	profileCalls int
	err          error
}

func (r *taskRepositoryStub) Get(context.Context, string) (users.User, error) {
	r.profileCalls++
	return users.User{Role: users.RoleIntern}, r.err
}

func (r *taskRepositoryStub) Tasks(context.Context, string) ([]Task, error) {
	r.calls++
	return nil, r.err
}

func (r *taskRepositoryStub) Create(
	_ context.Context,
	_, _ string,
	in CreateInput,
) (Task, bool, error) {
	r.input = in
	r.calls++

	return Task{Deadline: in.Deadline}, true, r.err
}

func (r *taskRepositoryStub) Complete(context.Context, string, string) (Task, error) {
	r.calls++
	return Task{}, r.err
}

func TestServiceBoundaries(t *testing.T) {
	repo := &taskRepositoryStub{}

	loc, locationErr := time.LoadLocation("Europe/Moscow")
	if locationErr != nil {
		t.Fatal(locationErr)
	}

	s := newTestService(
		repo,
		repo,
		loc,
		func() time.Time { return time.Date(2026, 11, 14, 21, 0, 0, 0, time.UTC) },
	)
	ctx := context.Background()
	in := CreateInput{Title: " Task ", Description: " Details ", Deadline: " 2026-11-14 "}

	task, _, err := s.CreateTask(ctx, "i", strings.Repeat("я", 128), in)
	if err != nil || !task.Overdue || repo.profileCalls != 0 ||
		repo.input != (CreateInput{Title: "Task", Description: "Details", Deadline: "2026-11-14"}) {
		t.Fatal(task, repo, err)
	}

	before := repo.calls

	for _, key := range []string{" ", strings.Repeat("я", 129)} {
		if _, _, err = s.CreateTask(ctx, "i", key, in); !errors.Is(err, ErrInvalid) {
			t.Fatal(err)
		}
	}

	for _, bad := range []CreateInput{{Title: " ", Deadline: "2026-11-14"}, {Title: "Task", Deadline: "2026-02-30"}} {
		if _, _, err = s.CreateTask(ctx, "i", "key", bad); !errors.Is(err, ErrInvalid) {
			t.Fatal(err)
		}
	}

	if repo.calls != before {
		t.Fatal("invalid input reached repository")
	}

	list, err := s.Progress(ctx, "i")
	if err != nil || list.Tasks == nil || repo.profileCalls != 1 {
		t.Fatal(list, err)
	}

	repo.err = ErrNotFound
	if _, err = s.CompleteTask(
		ctx,
		"missing",
		"task",
	); !errors.Is(err, ErrNotFound) ||
		repo.calls != before+2 {
		t.Fatal(err, repo)
	}
}

func TestPrepareListAndCalendar(t *testing.T) {
	loc, locationErr := time.LoadLocation("Europe/Moscow")
	if locationErr != nil {
		t.Fatal(locationErr)
	}

	before := time.Date(2026, 11, 14, 20, 59, 59, 0, time.UTC)
	if CalendarDate(before, loc) != "2026-11-14" ||
		CalendarDate(before.Add(time.Second), loc) != "2026-11-15" {
		t.Fatal("calendar midnight")
	}

	done := before
	list := []Task{
		{ID: "b", Deadline: "2026-11-14"},
		{ID: "a", Deadline: "2026-11-13", CompletedAt: &done},
	}
	service, repo := fixture(t)
	service.Now = func() time.Time { return before.Add(time.Second) }
	repo.snapshot.Tasks = list

	progress, err := service.Progress(context.Background(), "i")
	if err != nil {
		t.Fatal(err)
	}

	out := progress.Tasks
	if &out[0] != &list[0] || out[0].ID != "b" || !out[0].Overdue || out[1].Overdue ||
		out[1].Status != "completed" {
		t.Fatal(out)
	}

	repo.snapshot.Tasks = nil

	empty, err := service.Progress(context.Background(), "i")
	if err != nil || empty.Tasks == nil || len(empty.Tasks) != 0 {
		t.Fatal(empty, err)
	}
}

//nolint:nilnil // This fixture models an unassigned mentor.
func (*taskRepositoryStub) Mentor(
	context.Context,
	string,
) (*users.User, error) {
	return nil, nil
}

func (*taskRepositoryStub) Assigned(context.Context, string, string) (bool, error) {
	return false, nil
}

func (*taskRepositoryStub) Interns(
	context.Context,
	string,
) ([]users.User, error) {
	panic("unused")
}
