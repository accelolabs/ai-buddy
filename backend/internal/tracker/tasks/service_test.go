package tasks

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"ai-buddy/internal/tracker/apperror"
	"ai-buddy/internal/tracker/interns"
)

type repositoryStub struct {
	input        CreateInput
	calls        int
	profileCalls int
	err          error
}

func (r *repositoryStub) Intern(context.Context, string) (interns.Profile, error) {
	r.profileCalls++
	return interns.Profile{}, r.err
}
func (r *repositoryStub) Tasks(context.Context, string) ([]Task, error) { r.calls++; return nil, r.err }
func (r *repositoryStub) Create(_ context.Context, _, _ string, in CreateInput) (Task, bool, error) {
	r.input = in
	r.calls++
	return Task{Deadline: in.Deadline}, true, r.err
}
func (r *repositoryStub) Complete(context.Context, string, string) (Task, error) {
	r.calls++
	return Task{}, r.err
}

func TestServiceBoundaries(t *testing.T) {
	repo := &repositoryStub{}
	loc, _ := time.LoadLocation("Europe/Moscow")
	s := New(repo, repo, loc, func() time.Time { return time.Date(2026, 11, 14, 21, 0, 0, 0, time.UTC) })
	ctx := context.Background()
	in := CreateInput{Title: " Task ", Description: " Details ", Deadline: " 2026-11-14 "}
	task, _, err := s.Create(ctx, "i", strings.Repeat("я", 128), in)
	if err != nil || !task.Overdue || repo.profileCalls != 0 || repo.input != (CreateInput{Title: "Task", Description: "Details", Deadline: "2026-11-14"}) {
		t.Fatal(task, repo, err)
	}
	before := repo.calls
	for _, key := range []string{" ", strings.Repeat("я", 129)} {
		if _, _, err = s.Create(ctx, "i", key, in); !errors.Is(err, apperror.ErrInvalid) {
			t.Fatal(err)
		}
	}
	for _, bad := range []CreateInput{{Title: " ", Deadline: "2026-11-14"}, {Title: "Task", Deadline: "2026-02-30"}} {
		if _, _, err = s.Create(ctx, "i", "key", bad); !errors.Is(err, apperror.ErrInvalid) {
			t.Fatal(err)
		}
	}
	if repo.calls != before {
		t.Fatal("invalid input reached repository")
	}
	list, err := s.Tasks(ctx, "i")
	if err != nil || list == nil || repo.profileCalls != 1 {
		t.Fatal(list, err)
	}
	repo.err = apperror.ErrNotFound
	if _, err = s.Complete(ctx, "missing", "task"); !errors.Is(err, apperror.ErrNotFound) || repo.calls != before+1 {
		t.Fatal(err, repo)
	}
}

func TestPrepareListAndCalendar(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	before := time.Date(2026, 11, 14, 20, 59, 59, 0, time.UTC)
	if CalendarDate(before, loc) != "2026-11-14" || CalendarDate(before.Add(time.Second), loc) != "2026-11-15" {
		t.Fatal("calendar midnight")
	}
	done := before
	list := []Task{{ID: "b", Deadline: "2026-11-14"}, {ID: "a", Deadline: "2026-11-13", CompletedAt: &done}}
	out := PrepareList(list, "2026-11-15")
	if &out[0] != &list[0] || out[0].ID != "b" || !out[0].Overdue || out[1].Overdue || out[1].Status != "completed" {
		t.Fatal(out)
	}
	if empty := PrepareList(nil, "2026-11-15"); empty == nil || len(empty) != 0 {
		t.Fatal(empty)
	}
}
