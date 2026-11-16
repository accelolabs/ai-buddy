package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
					body := `{"description":"x","deadline":"2099-01-01T00:00:00Z"}`

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
