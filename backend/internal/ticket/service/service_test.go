package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	// Project packages.
	"backend/internal/ticket"
	"backend/internal/users"
)

type repositoryStub struct {
	value ticket.Ticket
	err   error
	calls int
	user  users.User
	id    string
}

func (r *repositoryStub) Add(_ context.Context, value ticket.Ticket) (ticket.Ticket, error) {
	r.calls++
	r.value = value

	return value, r.err
}

func (r *repositoryStub) List(_ context.Context, user users.User) ([]ticket.Ticket, error) {
	r.calls++
	r.user = user

	return nil, r.err
}

func (r *repositoryStub) Get(_ context.Context, user users.User, id string) (ticket.Ticket, error) {
	r.calls++
	r.user, r.id = user, id

	return r.value, r.err
}

func TestCreate(t *testing.T) {
	repo := &repositoryStub{}
	service := NewService(repo)
	user := users.User{ID: "trusted", Role: users.RoleIntern}
	first, err := service.Create(t.Context(), user, " \n Заголовок \t", " \t Описание \n")
	if err != nil ||
		first.ID == "" ||
		first.UserID != user.ID ||
		first.Title != "Заголовок" ||
		first.Description != "Описание" {
		t.Fatalf("created: %+v, %v", first, err)
	}

	second, err := service.Create(t.Context(), user, "Заголовок", "Описание")
	if err != nil || first.ID == second.ID {
		t.Fatalf("IDs: %q %q; %v", first.ID, second.ID, err)
	}

	for _, input := range [][2]string{{" ", "description"}, {"title", "\n "}} {
		before := repo.calls
		if _, err := service.Create(t.Context(), user, input[0], input[1]); !errors.Is(err, ErrInvalidInput) ||
			repo.calls != before {
			t.Fatalf("validation: %v", err)
		}
	}

	for _, user := range []users.User{{Role: users.RoleMentor}, {}} {
		before := repo.calls
		if _, err := service.Create(t.Context(), user, "", ""); !errors.Is(err, ErrForbidden) || repo.calls != before {
			t.Fatalf("role: %v", err)
		}
	}

	repo.err = fmt.Errorf("insert: %w", sql.ErrNoRows)
	if _, err := service.Create(t.Context(), user, "title", "description"); !errors.Is(err, sql.ErrNoRows) ||
		errors.Is(err, ErrNotFound) {
		t.Fatalf("insert error: %v", err)
	}
}

func TestRead(t *testing.T) {
	repo := &repositoryStub{}
	service := NewService(repo)

	for _, role := range []string{users.RoleIntern, users.RoleMentor} {
		user := users.User{ID: "trusted", Role: role}
		if _, err := service.List(t.Context(), user); err != nil || repo.user != user {
			t.Fatal("list identity")
		}

		if _, err := service.Get(t.Context(), user, "ticket"); err != nil || repo.user != user || repo.id != "ticket" {
			t.Fatal("read identity")
		}
	}

	repo.err = fmt.Errorf("read: %w", sql.ErrNoRows)
	if _, err := service.Get(t.Context(), users.User{}, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}

	repo.err = errors.New("unavailable")
	if _, err := service.Get(t.Context(), users.User{}, "ticket"); !errors.Is(err, repo.err) {
		t.Fatal(err)
	}

	if _, err := service.List(t.Context(), users.User{}); !errors.Is(err, repo.err) {
		t.Fatal(err)
	}
}
