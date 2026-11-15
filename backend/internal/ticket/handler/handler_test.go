package handler

import (
	"backend/internal/ticket"
	"backend/internal/users"
	"backend/internal/users/middleware"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

type userRepository struct{ role string }

func (s userRepository) Get(context.Context, string) (users.User, error) {
	return users.User{ID: "intern", Role: s.role}, nil
}

func TestValidation(t *testing.T) {
	for i, body := range []string{
		"", "null", "[]", "{", `{"title":"x"}`, `{"description":"x"}`, `{"title":" \t\n","description":"x"}`,
		`{"title":"x","description":"\u2003"}`, `{"title":1,"description":"x"}`, `{"title":"x","description":null}`,
		`{"title":"x","description":"y"} {}`, `{"title":"x","description":"y"} junk`,
		`{"title":"` + strings.Repeat("x", 1<<20) + `","description":"y"}`,
		`{"title":"x","description":"y"}` + strings.Repeat(" ", 1<<20),
	} {
		t.Run(fmt.Sprintf("body-%d", i), func(t *testing.T) {
			repo := &failingRepository{}
			h := NewHandler(repo)
			r := httptest.NewRequest("POST", "/api/tickets", strings.NewReader(body))
			r.AddCookie(&http.Cookie{Name: "demo_user", Value: "intern"})
			w := httptest.NewRecorder()
			middleware.Middleware(userRepository{users.RoleIntern})(http.HandlerFunc(h.Create)).ServeHTTP(w, r)
			if w.Code != 400 || repo.calls != 0 {
				t.Fatalf("validation: %d calls=%d", w.Code, repo.calls)
			}
		})
	}
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
			{users.RoleIntern, true, errors.New("private database details"), 500},
			{users.RoleMentor, true, errors.New("private database details"), 500},
			{users.RoleIntern, true, fmt.Errorf("wrapped: %w", sql.ErrNoRows), 404},
		} {
			t.Run(fmt.Sprintf("%s-%s-%d", method, tc.role, tc.status), func(t *testing.T) {
				repo := &failingRepository{err: tc.err}
				h := NewHandler(repo)
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
				r.AddCookie(&http.Cookie{Name: "demo_user", Value: "intern"})
				w := httptest.NewRecorder()
				target.ServeHTTP(w, r)
				want := tc.status
				if tc.selected && method == "create" && tc.role == users.RoleMentor {
					want = 403
				} else if tc.status == 404 && method != "get" {
					want = 500
				}
				if w.Code != want || strings.Contains(w.Body.String(), "private") || strings.Contains(w.Body.String(), "wrapped") {
					t.Fatalf("error: %d %s", w.Code, w.Body)
				}
				if (!tc.selected || want == 403) && repo.calls != 0 {
					t.Fatal("unauthorized repository call")
				}
			})
		}
	}
}
