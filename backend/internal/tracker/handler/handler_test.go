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
	"backend/internal/tracker"
	"backend/internal/tracker/service"
	"backend/internal/users"
	"backend/internal/users/middleware"
)

type failingRepository struct {
	calls int
	err   error
}

func (s *failingRepository) Add(context.Context, tracker.Task) (tracker.Task, error) {
	s.calls++

	return tracker.Task{}, s.err
}

func (s *failingRepository) List(context.Context, users.User, string) ([]tracker.Task, error) {
	s.calls++

	return nil, s.err
}

func (s *failingRepository) Complete(context.Context, users.User, string) (tracker.Task, error) {
	s.calls++

	return tracker.Task{}, s.err
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
	for _, method := range []string{"create", "list", "intern", "complete"} {
		role := users.RoleIntern
		otherRole := users.RoleMentor
		if method == "intern" {
			role, otherRole = otherRole, role
		}

		for _, tc := range []struct {
			role     string
			selected bool
			err      error
			status   int
		}{
			{role, false, nil, 401},
			{otherRole, true, nil, 403},
			{role, true, errors.New("private database details"), 500},
			{role, true, fmt.Errorf("wrapped: %w", sql.ErrNoRows), 404},
		} {
			t.Run(
				fmt.Sprintf("%s-%d", method, tc.status),
				func(t *testing.T) {
					repo := &failingRepository{err: tc.err}
					h := NewHandler(service.NewService(repo))
					f := h.Create
					body := `{"description":"x","deadline":"2099-01-01"}`

					switch method {
					case "list":
						f = h.List
					case "intern":
						f = h.ListIntern
					case "complete":
						f = h.Complete
						body = ""
					}

					var target http.Handler = http.HandlerFunc(f)
					if tc.selected {
						target = middleware.Middleware(userRepository{tc.role})(target)
					}

					r := httptest.NewRequest("POST", "/", strings.NewReader(body))
					r.AddCookie(&http.Cookie{
						Name:  "demo_user",
						Value: "intern",
					})
					w := httptest.NewRecorder()
					target.ServeHTTP(w, r)
					want := tc.status
					if method == "create" && want == 404 {
						want = 500
					}

					if w.Code != want || strings.Contains(w.Body.String(), "private") || strings.Contains(w.Body.String(), "wrapped") {
						t.Fatalf("error: %d %s", w.Code, w.Body)
					}

					if (want == 401 || want == 403) && repo.calls != 0 {
						t.Fatal("unauthorized repository call")
					}
				},
			)
		}
	}
}

type memoryRepository struct {
	values []tracker.Task
	calls  int
}

func (s *memoryRepository) Add(_ context.Context, value tracker.Task) (tracker.Task, error) {
	s.calls++
	value.CreatedAt = time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC)
	value.Status = "pending"
	s.values = append(s.values, value)

	return value, nil
}

func (s *memoryRepository) List(_ context.Context, user users.User, internID string) ([]tracker.Task, error) {
	result := []tracker.Task{}

	for _, v := range s.values {
		if v.UserID == internID {
			result = append(result, v)
		}
	}

	return result, nil
}

func (s *memoryRepository) Complete(_ context.Context, user users.User, id string) (tracker.Task, error) {
	for i := range s.values {
		if s.values[i].UserID == user.ID && s.values[i].ID == id {
			s.values[i].Status = "completed"

			return s.values[i], nil
		}
	}

	return tracker.Task{}, sql.ErrNoRows
}

func TestHTTPValidationAndResponses(t *testing.T) {
	repo := &memoryRepository{}
	h := NewHandler(service.NewService(repo))
	mux := http.NewServeMux()
	selected := middleware.Middleware(userRepository{users.RoleIntern})
	mux.Handle("POST /api/tracker/tasks", selected(http.HandlerFunc(h.Create)))
	mux.Handle("GET /api/tracker/tasks", selected(http.HandlerFunc(h.List)))
	mux.Handle("POST /api/tracker/tasks/{id}/complete", selected(http.HandlerFunc(h.Complete)))
	mentor := middleware.Middleware(userRepository{users.RoleMentor})
	mux.Handle("GET /api/tracker/interns/{internID}/tasks", mentor(http.HandlerFunc(h.ListIntern)))
	request := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: "demo_user", Value: "intern"})
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		return w
	}

	w := request("GET", "/api/tracker/tasks", "")
	if w.Code != 200 || w.Body.String() != "[]\n" {
		t.Fatalf("empty %d %s", w.Code, w.Body)
	}

	for _, body := range []string{
		"", "null", "{}", "{", `{,
} {}`, `{} x`, `{"description":"` + strings.Repeat("x", 1<<20) + `"}`,
	} {
		w := request("POST", "/api/tracker/tasks", body)
		if w.Code != 400 || repo.calls != 0 {
			t.Fatalf("invalid %d %s; calls %d", w.Code, w.Body, repo.calls)
		}
	}

	for range 2 {
		w := request("POST", "/api/tracker/tasks", `{"description":" Body ",`+
			`"deadline":"2030-01-02","user_id":"forged"}`)
		var value taskResponse
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

	w = request("GET", "/api/tracker/tasks", "")
	var values []taskResponse
	if err := json.Unmarshal(w.Body.Bytes(), &values); err != nil {
		t.Fatal(err)
	}

	if w.Code != 200 || len(values) != 2 {
		t.Fatalf("list %d %s", w.Code, w.Body)
	}

	w = request("GET", "/api/tracker/interns/intern/tasks", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), repo.values[0].ID) {
		t.Fatalf("intern list %d %s", w.Code, w.Body)
	}

	for _, body := range []string{" ", "{}", "null"} {
		w = request("POST", "/api/tracker/tasks/"+repo.values[0].ID+"/complete", body)
		if w.Code != 400 || repo.values[0].Status != "pending" {
			t.Fatalf("completion body %d %s", w.Code, w.Body)
		}
	}

	for range 2 {
		w = request("POST", "/api/tracker/tasks/"+repo.values[0].ID+"/complete", "")
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"status":"completed"`) {
			t.Fatalf("complete %d %s", w.Code, w.Body)
		}
	}
}
