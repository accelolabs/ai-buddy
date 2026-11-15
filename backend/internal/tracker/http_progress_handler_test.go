package tracker

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-buddy/internal/users"
)

type progressStub struct {
	err          error
	id, mentorID string
}

func (s *progressStub) Progress(_ context.Context, id string) (Progress, error) {
	s.id = id
	return Progress{Intern: users.User{ID: id, Role: users.RoleIntern}}, s.err
}

func (s *progressStub) MentorProgress(_ context.Context, mentor, id string) (Progress, error) {
	s.id = id
	s.mentorID = mentor
	return Progress{Intern: users.User{ID: id, Role: users.RoleIntern}}, s.err
}

func mentorRequest(t *testing.T, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("GET", "/api/interns/i/progress", nil)
	r.SetPathValue("id", "i")
	r = r.WithContext(users.WithUser(r.Context(), users.User{ID: "m", Role: "mentor", Team: "backend"}))
	w := httptest.NewRecorder()
	handler(w, r)

	return w
}

func TestProgressHandlers(t *testing.T) {
	service := &progressStub{}
	h := &Service{Profiles: service, Store: service, Location: time.UTC, Now: time.Now}
	get := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/api/progress", nil)
		r = r.WithContext(users.WithUser(r.Context(), users.User{ID: "i", Role: "intern", Team: "backend"}))
		w := httptest.NewRecorder()
		h.ProgressHTTP(w, r)
		return w
	}
	w := get()
	if w.Code != 200 || service.id != "i" {
		t.Fatal(w, service)
	}

	w = mentorRequest(t, h.MentorProgressHTTP)
	if w.Code != 200 || service.id != "i" || service.mentorID != "m" {
		t.Fatal(w, service)
	}

	for _, tc := range []struct {
		err  error
		code int
	}{{ErrNotFound, 404}, {errors.New("SQL secret"), 500}} {
		service.err = tc.err
		for _, w := range []*httptest.ResponseRecorder{get(), mentorRequest(t, h.MentorProgressHTTP)} {
			if w.Code != tc.code || strings.Contains(w.Body.String(), "secret") {
				t.Fatal(w)
			}
		}
	}
}

func (s *progressStub) Get(_ context.Context, id string) (users.User, error) {
	s.id = id
	return users.User{ID: id, Role: users.RoleIntern}, s.err
}
func (s *progressStub) Mentor(context.Context, string) (*users.User, error) { return nil, nil }
func (s *progressStub) Assigned(_ context.Context, mentor, id string) (bool, error) {
	s.mentorID = mentor
	s.id = id
	return true, s.err
}
func (s *progressStub) Interns(context.Context, string) ([]users.User, error) { panic("unused") }
func (s *progressStub) Tasks(context.Context, string) ([]Task, error)         { return []Task{}, nil }
func (s *progressStub) Create(context.Context, string, string, CreateInput) (Task, bool, error) {
	panic("unused")
}
func (s *progressStub) Complete(context.Context, string, string) (Task, error) { panic("unused") }
