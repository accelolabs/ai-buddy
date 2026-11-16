package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	// Project packages.
	"backend/internal/ticket"
	"backend/internal/ticket/service"
	"backend/internal/users"
	"backend/internal/users/middleware"
)

type failingRepository struct {
	calls int
	err   error
}

func (s *failingRepository) Add(context.Context, ticket.Ticket) (ticket.Ticket, error) {
	s.calls++

	return ticket.Ticket{}, s.err
}

func (s *failingRepository) List(context.Context, users.User) ([]ticket.Ticket, error) {
	s.calls++

	return nil, s.err
}

func (s *failingRepository) Get(context.Context, users.User, string) (ticket.Ticket, error) {
	s.calls++

	return ticket.Ticket{}, s.err
}

type userRepository struct {
	role string
}

func (s userRepository) Get(context.Context, string) (users.User, error) {
	return users.User{
		ID:   "intern",
		Role: s.role,
	}, nil
}

func TestErrors(t *testing.T) {
	for _, method := range []string{"create", "list", "get"} {
		for _, tc := range []struct {
			role     string
			selected bool
			err      error
			status   int
		}{
			{users.RoleIntern, false, nil, 401},
			{users.RoleMentor, true, nil, 403},
			{users.RoleIntern, true, errors.New("private database details"), 500},
			{users.RoleMentor, true, errors.New("private database details"), 500},
			{users.RoleIntern, true, fmt.Errorf("wrapped: %w", sql.ErrNoRows), 404},
		} {
			if tc.status == 403 && method != "create" {
				continue
			}

			t.Run(
				fmt.Sprintf("%s-%s-%d", method, tc.role, tc.status),
				func(t *testing.T) {
					repo := &failingRepository{err: tc.err}
					h := NewHandler(service.NewService(repo))
					f := h.Get

					switch method {
					case "create":
						f = h.Create
					case "list":
						f = h.List
					}

					var target http.Handler = http.HandlerFunc(f)
					if tc.selected {
						target = middleware.Middleware(userRepository{tc.role})(target)
					}

					r := httptest.NewRequest("POST", "/api/tickets", strings.NewReader(`{"title":"x","description":"y"}`))
					r.AddCookie(&http.Cookie{
						Name:  "demo_user",
						Value: "intern",
					})
					w := httptest.NewRecorder()
					target.ServeHTTP(w, r)
					want := tc.status
					if tc.status == 404 && method != "get" {
						want = 500
					}

					if method == "create" && tc.role == users.RoleMentor {
						want = 403
					}

					if w.Code != want || strings.Contains(w.Body.String(), "private") || strings.Contains(w.Body.String(), "wrapped") {
						t.Fatalf("error: %d %s", w.Code, w.Body)
					}

					if (!tc.selected || want == 403) && repo.calls != 0 {
						t.Fatal("unauthorized repository call")
					}
				},
			)
		}
	}
}

type memoryRepository struct {
	values []ticket.Ticket
	calls  int
}

func (s *memoryRepository) Add(_ context.Context, value ticket.Ticket) (ticket.Ticket, error) {
	s.calls++
	value.CreatedAt = time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC)
	s.values = append(s.values, value)

	return value, nil
}

func (s *memoryRepository) List(_ context.Context, user users.User) ([]ticket.Ticket, error) {
	result := []ticket.Ticket{}

	for _, v := range s.values {
		if v.UserID == user.ID {
			result = append(result, v)
		}
	}

	return result, nil
}

func (s *memoryRepository) Get(_ context.Context, user users.User, id string) (ticket.Ticket, error) {
	for _, v := range s.values {
		if v.UserID == user.ID && v.ID == id {
			return v, nil
		}
	}

	return ticket.Ticket{}, sql.ErrNoRows
}

func TestHTTPValidationAndResponses(t *testing.T) {
	repo := &memoryRepository{}
	h := NewHandler(service.NewService(repo))
	mux := http.NewServeMux()
	selected := middleware.Middleware(userRepository{users.RoleIntern})
	mux.Handle("POST /api/tickets", selected(http.HandlerFunc(h.Create)))
	mux.Handle("GET /api/tickets", selected(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/tickets/{id}", selected(http.HandlerFunc(h.Get)))
	request := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: "demo_user", Value: "intern"})
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		return w
	}

	w := request("GET", "/api/tickets", "")
	if w.Code != 200 || w.Body.String() != "[]\n" {
		t.Fatalf("empty %d %s", w.Code, w.Body)
	}

	for _, body := range []string{
		"", "null", "{}", "{", `{,
} {}`, `{} x`, `{"description":"` + strings.Repeat("x", 1<<20) + `"}`,
	} {
		w := request("POST", "/api/tickets", body)
		if w.Code != 400 || repo.calls != 0 {
			t.Fatalf("invalid %d %s; calls %d", w.Code, w.Body, repo.calls)
		}
	}

	for range 2 {
		w := request("POST", "/api/tickets", `{"title":" Title ","description":" Body ","user_id":"forged"}`)
		var value ticketResponse
		if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
			t.Fatal(err)
		}

		if w.Code != 201 || value.UserID != "intern" || value.Description != "Body" ||
			value.ID == "" || value.CreatedAt.IsZero() {
			t.Fatalf("created %d %+v", w.Code, value)
		}
	}

	if repo.values[0].ID == repo.values[1].ID {
		t.Fatal("duplicate IDs")
	}

	w = request("GET", "/api/tickets", "")
	var values []ticketResponse
	if err := json.Unmarshal(w.Body.Bytes(), &values); err != nil {
		t.Fatal(err)
	}

	if w.Code != 200 || len(values) != 2 {
		t.Fatalf("list %d %s", w.Code, w.Body)
	}

	w = request("GET", "/api/tickets/"+repo.values[0].ID, "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"title":"Title"`) {
		t.Fatalf("get %d %s", w.Code, w.Body)
	}
}
