package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	// Project packages.
	"backend/internal/tracker"
	"backend/internal/users"
)

type repositoryStub struct {
	value tracker.Task
	err   error
	calls int
	user  users.User
	id    string
}

func (r *repositoryStub) Add(_ context.Context, value tracker.Task) (tracker.Task, error) {
	r.calls++
	r.value = value
	return value, r.err
}

func (r *repositoryStub) List(_ context.Context, user users.User, id string) ([]tracker.Task, error) {
	r.calls++
	r.user, r.id = user, id
	return nil, r.err
}

func (r *repositoryStub) Complete(_ context.Context, user users.User, id string) (tracker.Task, error) {
	r.calls++
	r.user, r.id = user, id
	return r.value, r.err
}

func TestCreate(t *testing.T) {
	repo := &repositoryStub{}
	service := NewService(repo)
	user := users.User{ID: "trusted", Role: users.RoleIntern}
	first, err := service.Create(t.Context(), user, " \n задача \t", "2000-01-01T12:00:00+03:00")
	if err != nil ||
		first.ID == "" ||
		first.UserID != user.ID ||
		first.Description != "задача" ||
		first.Deadline.Format("2006-01-02T15:04:05Z07:00") != "2000-01-01T12:00:00+03:00" {
		t.Fatalf("created: %+v, %v", first, err)
	}
	second, err := service.Create(t.Context(), user, "задача", "2000-01-01T12:00:00+03:00")
	if err != nil || first.ID == second.ID {
		t.Fatalf("IDs: %q %q; %v", first.ID, second.ID, err)
	}
	for _, input := range [][2]string{
		{" ", "2000-01-01T00:00:00Z"},
		{"task", ""},
		{"task", "2000-01-01T00:00:00"},
		{"task", "invalid"},
	} {
		before := repo.calls
		if _, err := service.Create(t.Context(), user, input[0], input[1]); !errors.Is(err, ErrInvalidInput) ||
			repo.calls != before {
			t.Fatalf("validation: %v", err)
		}
	}
	repo.err = fmt.Errorf("insert: %w", sql.ErrNoRows)
	if _, err := service.Create(t.Context(), user, "task", "2000-01-01T00:00:00Z"); !errors.Is(err, sql.ErrNoRows) ||
		errors.Is(err, ErrNotFound) {
		t.Fatalf("insert error: %v", err)
	}
}

func TestAccessAndErrors(t *testing.T) {
	repo := &repositoryStub{}
	service := NewService(repo)
	intern := users.User{ID: "intern", Role: users.RoleIntern}
	mentor := users.User{ID: "mentor", Role: users.RoleMentor}
	for _, user := range []users.User{mentor, {}} {
		if _, err := service.Create(t.Context(), user, "", ""); !errors.Is(err, ErrForbidden) {
			t.Fatal(err)
		}
		if _, err := service.List(t.Context(), user); !errors.Is(err, ErrForbidden) {
			t.Fatal(err)
		}
		if _, err := service.Complete(t.Context(), user, "task"); !errors.Is(err, ErrForbidden) {
			t.Fatal(err)
		}
	}

	if _, err := service.ListIntern(t.Context(), intern, "other"); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}

	if repo.calls != 0 {
		t.Fatal("forbidden calls reached repository")
	}

	if _, err := service.List(t.Context(), intern); err != nil || repo.user != intern || repo.id != intern.ID {
		t.Fatal("own list identity")
	}

	if _, err := service.ListIntern(t.Context(), mentor, "selected"); err != nil ||
		repo.user != mentor ||
		repo.id != "selected" {
		t.Fatal("mentor list identity")
	}

	if _, err := service.Complete(t.Context(), intern, "task"); err != nil || repo.user != intern || repo.id != "task" {
		t.Fatal("completion identity")
	}
	for _, cause := range []error{fmt.Errorf("read: %w", sql.ErrNoRows), errors.New("database unavailable")} {
		repo.err = cause
		for _, call := range []func() error{
			func() error { _, err := service.List(t.Context(), intern); return err },
			func() error { _, err := service.ListIntern(t.Context(), mentor, "selected"); return err },
			func() error { _, err := service.Complete(t.Context(), intern, "task"); return err },
		} {
			err := call()
			if errors.Is(cause, sql.ErrNoRows) {
				if !errors.Is(err, ErrNotFound) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, cause) {
				t.Fatal(err)
			}
		}
	}
}
