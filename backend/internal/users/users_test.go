package users

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type profiles struct{ err error }

func (p profiles) Get(_ context.Context, id string) (User, error) {
	if p.err != nil {
		return User{}, p.err
	}
	if id != "intern.backend.1" {
		return User{}, sql.ErrNoRows
	}
	return User{ID: id, Role: RoleIntern, Team: "backend", Username: id}, nil
}
func (p profiles) List(context.Context) ([]User, error) { return []User{}, p.err }
func TestSelection(t *testing.T) {
	for _, tc := range []struct {
		cookie string
		err    error
		status int
	}{{"", nil, 400}, {"demo_user=", nil, 400}, {"demo_user=unknown", nil, 400}, {"demo_user=intern.backend.1", nil, 200}, {"demo_user=intern.backend.1", errors.New("SQL secret"), 503}} {
		mux := http.NewServeMux()
		s := &Selector{Profiles: profiles{err: tc.err}}
		s.Register(mux)
		r := httptest.NewRequest("GET", "/api/me", nil)
		r.Header.Set("Cookie", tc.cookie)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != tc.status || !strings.Contains(w.Header().Get("Content-Type"), "application/json") || strings.Contains(w.Body.String(), "secret") {
			t.Fatal(w)
		}
	}
	mux := http.NewServeMux()
	s := &Selector{Profiles: profiles{}}
	s.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/users", nil))
	if w.Code != 200 || w.Body.String() != "{\"users\":[]}\n" {
		t.Fatal(w)
	}
	w = httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "demo_user", Value: "intern.backend.1"})
	s.RequireRoles(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("mentor handler called") }), RoleMentor).ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w)
	}
}
