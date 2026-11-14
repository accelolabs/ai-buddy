package tasks

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"ai-buddy/backend-tracker/internal/apperror"
	"ai-buddy/backend-tracker/internal/interns"
)

type repositoryStub struct {
	list                  []Task
	input                 CreateInput
	key, internID, taskID string
	err                   error
	created               bool
}

func (r *repositoryStub) Tasks(_ context.Context, id string) ([]Task, error) {
	r.internID = id
	return r.list, r.err
}

func (r *repositoryStub) Create(_ context.Context, id, key string, input CreateInput) (Task, bool, error) {
	r.internID = id
	r.key = key
	r.input = input
	return Task{
		ID:          "task",
		InternID:    id,
		Title:       input.Title,
		Description: input.Description,
		Deadline:    input.Deadline,
	}, r.created, r.err
}

func (r *repositoryStub) Complete(_ context.Context, id, taskID string) (Task, error) {
	r.internID = id
	r.taskID = taskID
	return Task{ID: taskID, InternID: id}, r.err
}

type internStub struct {
	err error
	id  string
}

func (i *internStub) Intern(_ context.Context, id string) (interns.Profile, error) {
	i.id = id
	return interns.Profile{ID: id}, i.err
}

func newService() (*Service, *repositoryStub, *internStub) {
	repo := &repositoryStub{created: true}
	intern := &internStub{}
	loc, _ := time.LoadLocation("Europe/Moscow")
	return New(repo, intern, loc, func() time.Time {
		return time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	}), repo, intern
}

func TestCreate(t *testing.T) {
	s, repo, intern := newService()
	ctx := context.Background()
	input := CreateInput{
		Title:       "  Task  ",
		Description: " Details ",
		Deadline:    " 2020-01-01 ",
	}
	task, created, err := s.Create(ctx, "i", "key", input)
	if err != nil ||
		!created ||
		!task.Overdue ||
		repo.input != (CreateInput{
			Title:       "Task",
			Description: "Details",
			Deadline:    "2020-01-01",
		}) ||
		repo.key != "key" ||
		repo.internID != "i" ||
		intern.id != "i" {
		t.Fatal(task, created, repo, intern, err)
	}

	repo.created = false
	task, created, err = s.Create(ctx, "i", "key", input)
	if err != nil || created || task.ID != "task" {
		t.Fatal(task, created, err)
	}

	for _, key := range []string{"", "  ", strings.Repeat("k", 257)} {
		if _, _, err = s.Create(ctx, "i", key, input); err != apperror.ErrInvalid {
			t.Fatal(key, err)
		}
	}

	for _, in := range []CreateInput{
		{Title: "  ", Deadline: "2026-01-01"},
		{Title: "a", Deadline: "2026-02-30"},
		{Title: "a", Deadline: "2026-1-01"},
		{Title: "a"},
	} {
		if _, _, err = s.Create(ctx, "i", "k", in); err != apperror.ErrInvalid {
			t.Fatal(in, err)
		}
	}
}

func TestTasksAndComplete(t *testing.T) {
	s, repo, intern := newService()
	ctx := context.Background()
	list, err := s.Tasks(ctx, "i")
	if err != nil || list == nil || len(list) != 0 || intern.id != "i" {
		t.Fatal(list, err)
	}

	now := s.now()
	done := now
	repo.list = []Task{
		{
			ID:        "z",
			Deadline:  "2026-11-13",
			CreatedAt: now.Add(time.Hour),
		},
		{
			ID:        "b",
			Deadline:  "2026-11-14",
			CreatedAt: now,
		},
		{
			ID:        "a",
			Deadline:  "2026-11-13",
			CreatedAt: now,
		},
		{
			ID:          "c",
			Deadline:    "2026-11-01",
			CreatedAt:   now,
			CompletedAt: &done,
		},
	}
	list, err = s.Tasks(ctx, "i")
	if err != nil ||
		list[0].ID != "a" ||
		!list[0].Overdue ||
		list[1].ID != "b" ||
		list[1].Overdue ||
		list[2].Status != "completed" ||
		list[2].Overdue ||
		list[3].ID != "z" {
		t.Fatal(list, err)
	}

	task, err := s.Complete(ctx, "i", "task")
	if err != nil || task.ID != "task" || repo.internID != "i" || repo.taskID != "task" {
		t.Fatal(task, err)
	}
}

func TestCalendarBoundary(t *testing.T) {
	s, repo, _ := newService()
	repo.list = []Task{{Deadline: "2026-11-14"}}
	calls := 0
	s.now = func() time.Time {
		calls++
		return time.Date(2026, 11, 14, 20, 59, 59, 0, time.UTC)
	}
	list, _ := s.Tasks(context.Background(), "i")
	if list[0].Overdue || calls != 1 {
		t.Fatal(list, calls)
	}

	s.now = func() time.Time {
		return time.Date(2026, 11, 14, 21, 0, 0, 0, time.UTC)
	}
	list, _ = s.Tasks(context.Background(), "i")
	if !list[0].Overdue {
		t.Fatal("local midnight", list)
	}
}

func TestDependencyErrors(t *testing.T) {
	for _, source := range []string{"intern", "repository"} {
		for _, want := range []error{apperror.ErrNotFound, apperror.ErrConflict, errors.New("database error")} {
			t.Run(source+want.Error(), func(t *testing.T) {
				s, repo, intern := newService()
				if source == "intern" {
					intern.err = want
				} else {
					repo.err = want
				}

				ctx := context.Background()
				if _, err := s.Tasks(ctx, "i"); err != want {
					t.Fatal(err)
				}

				if _, _, err := s.Create(ctx, "i", "k", CreateInput{Title: "a", Deadline: "2026-01-01"}); err != want {
					t.Fatal(err)
				}

				if _, err := s.Complete(ctx, "i", "task"); err != want {
					t.Fatal(err)
				}

				if source == "intern" && repo.internID != "" {
					t.Fatal("repository called after failed profile check")
				}
			})
		}
	}
}

func TestPrepareListPreservesInput(t *testing.T) {
	now := time.Now()
	original := []Task{
		{
			ID:        "z",
			CreatedAt: now,
			Deadline:  "2026-11-13",
		},
		{
			ID:        "a",
			CreatedAt: now,
			Deadline:  "2026-11-14",
		},
	}
	prepared := PrepareList(original, "2026-11-14")
	if original[0].ID != "z" ||
		original[0].Status != "" ||
		original[0].Overdue ||
		prepared[0].ID != "a" ||
		!prepared[1].Overdue {
		t.Fatal(original, prepared)
	}

	prepared[0].Title = "Changed"
	if original[1].Title != "" {
		t.Fatal("result aliases source")
	}
}
