package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	// Project packages.
	"backend/internal/users"
	"backend/internal/users/middleware"
)

type listFunc func(context.Context) ([]users.User, error)

func (f listFunc) List(ctx context.Context) ([]users.User, error) {
	return f(ctx)
}

func (f listFunc) ListInterns(ctx context.Context, _ string) ([]users.User, error) {
	return f(ctx)
}

func TestHandlerList(t *testing.T) {
	tests := []struct {
		name   string
		users  []users.User
		err    error
		status int
		body   string
	}{
		{
			name: "users",
			users: []users.User{
				{
					ID:   "intern.backend.1",
					Name: "Фёдор Добронравов",
					Role: users.RoleIntern,
					Team: "backend",
				},
				{
					ID:   "mentor.backend",
					Name: "Сергей Дорогов",
					Role: users.RoleMentor,
					Team: "backend",
				},
			},
			status: http.StatusOK,
			body: "[{\"id\":\"intern.backend.1\",\"name\":\"Фёдор" +
				" Добронравов\",\"role\":\"intern\",\"team\":\"backend\"}," +
				`{"id":"mentor.backend","name":"Сергей Дорогов","role":"mentor","team":"backend"}]` + "\n",
		},
		{
			name:   "empty list",
			status: http.StatusOK,
			body:   "[]\n",
		},
		{
			name:   "repository error",
			err:    errors.New("private database error"),
			status: http.StatusInternalServerError,
			body:   "failed to list users\n",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
				handler := NewHandler(listFunc(func(ctx context.Context) ([]users.User, error) {
					if ctx != req.Context() {
						t.Error("repository did not receive request context")
					}

					return tt.users, tt.err
				}))
				mux := http.NewServeMux()
				mux.HandleFunc("GET /api/users", handler.List)
				recorder := httptest.NewRecorder()
				mux.ServeHTTP(recorder, req)

				if recorder.Code != tt.status || recorder.Body.String() != tt.body {
					t.Fatalf("response = %d %q, want %d %q", recorder.Code, recorder.Body.String(), tt.status, tt.body)
				}

				if tt.status == http.StatusOK && recorder.Header().Get("Content-Type") != "application/json" {
					t.Fatalf("Content-Type = %q, want application/json", recorder.Header().Get("Content-Type"))
				}
			},
		)
	}
}

type selectedUserRepository struct {
	role string
}

func (s selectedUserRepository) Get(context.Context, string) (users.User, error) {
	return users.User{
		ID:   "mentor",
		Role: s.role,
	}, nil
}

type internLister struct {
	mentorID string
	err      error
}

func (s *internLister) List(context.Context) ([]users.User, error) {
	return nil, nil
}

func (s *internLister) ListInterns(_ context.Context, id string) ([]users.User, error) {
	s.mentorID = id

	return nil, s.err
}

func TestListInterns(t *testing.T) {
	for _, tc := range []struct {
		role     string
		selected bool
		err      error
		status   int
	}{
		{users.RoleMentor, false, nil, 401},
		{users.RoleIntern, true, nil, 403},
		{users.RoleMentor, true, errors.New("private details"), 500},
		{users.RoleMentor, true, nil, 200},
	} {
		repo := &internLister{err: tc.err}
		h := NewHandler(repo)

		var target http.Handler = http.HandlerFunc(h.ListInterns)
		if tc.selected {
			target = middleware.Middleware(selectedUserRepository{tc.role})(target)
		}

		r := httptest.NewRequest("GET", "/api/users/interns", nil)
		r.AddCookie(&http.Cookie{
			Name:  "demo_user",
			Value: "mentor",
		})
		w := httptest.NewRecorder()
		target.ServeHTTP(w, r)
		if w.Code != tc.status || strings.Contains(w.Body.String(), "private") {
			t.Fatalf("response: %d %s", w.Code, w.Body)
		}

		if tc.status == 200 && (repo.mentorID != "mentor" || w.Body.String() != "[]\n") {
			t.Fatalf("empty list: %s %q", repo.mentorID, w.Body)
		}

		if (tc.status == 401 || tc.status == 403) && repo.mentorID != "" {
			t.Fatal("unauthorized repository call")
		}
	}
}
