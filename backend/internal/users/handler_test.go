package users

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type listFunc func(context.Context) ([]User, error)

func (f listFunc) List(ctx context.Context) ([]User, error) {
	return f(ctx)
}

func TestHandlerList(t *testing.T) {
	tests := []struct {
		name   string
		users  []User
		err    error
		status int
		body   string
	}{
		{
			name: "users",
			users: []User{
				{ID: "intern.backend.1", Name: "Фёдор Добронравов", Role: RoleIntern, Team: "backend"},
				{ID: "mentor.backend", Name: "Сергей Дорогов", Role: RoleMentor, Team: "backend"},
			},
			status: http.StatusOK,
			body: `[{"id":"intern.backend.1","name":"Фёдор Добронравов","role":"intern","team":"backend"},` +
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
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
			handler := NewHandler(listFunc(func(ctx context.Context) ([]User, error) {
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
		})
	}
}
