package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	// Project packages.
	"backend/internal/users"
)

type repositoryStub struct {
	calls int
	id    string
	err   error
	value users.User
}

func (r *repositoryStub) List(context.Context) ([]users.User, error) {
	r.calls++
	return []users.User{r.value}, r.err
}

func (r *repositoryStub) ListInterns(_ context.Context, id string) ([]users.User, error) {
	r.calls++
	r.id = id
	return []users.User{r.value}, r.err
}

func (r *repositoryStub) Get(_ context.Context, id string) (users.User, error) {
	r.calls++
	r.id = id
	return r.value, r.err
}

func (r *repositoryStub) GetMentor(ctx context.Context, id string) (users.User, error) {
	return r.Get(ctx, id)
}

func TestService(t *testing.T) {
	repo := &repositoryStub{value: users.User{ID: "result"}}
	service := NewService(repo)
	for _, role := range []string{users.RoleIntern, ""} {
		if _, err := service.ListInterns(t.Context(), users.User{Role: role}); !errors.Is(err, ErrForbidden) {
			t.Fatal(err)
		}
	}

	if repo.calls != 0 {
		t.Fatal("forbidden call reached repository")
	}

	if values, err := service.ListInterns(t.Context(), users.User{ID: "trusted", Role: users.RoleMentor}); err != nil ||
		len(values) != 1 ||
		values[0] != repo.value ||
		repo.id != "trusted" {
		t.Fatalf("interns: %v; %v", values, err)
	}

	if values, err := service.List(t.Context()); err != nil || len(values) != 1 || values[0] != repo.value {
		t.Fatalf("list: %v; %v", values, err)
	}
	for _, read := range []func(context.Context, string) (users.User, error){service.Get, service.GetMentor} {
		value, err := read(t.Context(), "selected")
		if err != nil || value != repo.value || repo.id != "selected" {
			t.Fatalf("get: %+v; %v", value, err)
		}
		repo.err = fmt.Errorf("read: %w", sql.ErrNoRows)
		if _, err := read(t.Context(), "missing"); !errors.Is(err, ErrNotFound) {
			t.Fatal(err)
		}
		repo.err = errors.New("unavailable")
		if _, err := read(t.Context(), "selected"); !errors.Is(err, repo.err) {
			t.Fatal(err)
		}
		repo.err = nil
	}
	repo.err = errors.New("list failure")
	if _, err := service.List(t.Context()); !errors.Is(err, repo.err) {
		t.Fatal(err)
	}

	if _, err := service.ListInterns(t.Context(), users.User{Role: users.RoleMentor}); !errors.Is(err, repo.err) {
		t.Fatal(err)
	}
}
