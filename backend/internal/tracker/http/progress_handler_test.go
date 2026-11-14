package trackerhttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-buddy/internal/tracker/apperror"
	"ai-buddy/internal/tracker/interns"
	"ai-buddy/internal/tracker/progress"
	"ai-buddy/internal/users"
)

type progressStub struct {
	err          error
	id, mentorID string
}

func (s *progressStub) Progress(_ context.Context, id string) (progress.Progress, error) {
	s.id = id
	return progress.Progress{Intern: interns.Profile{ID: id}}, s.err
}

func (s *progressStub) MentorProgress(_ context.Context, mentor, id string) (progress.Progress, error) {
	s.id = id
	s.mentorID = mentor
	return progress.Progress{Intern: interns.Profile{ID: id}}, s.err
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
	h := New(nil, nil, service)
	get := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/api/progress", nil)
		r = r.WithContext(users.WithUser(r.Context(), users.User{ID: "i", Role: "intern", Team: "backend"}))
		w := httptest.NewRecorder()
		h.Progress(w, r)
		return w
	}
	w := get()
	if w.Code != 200 || service.id != "i" {
		t.Fatal(w, service)
	}

	w = mentorRequest(t, h.MentorProgress)
	if w.Code != 200 || service.id != "i" || service.mentorID != "m" {
		t.Fatal(w, service)
	}

	for _, tc := range []struct {
		err  error
		code int
	}{{apperror.ErrNotFound, 404}, {errors.New("SQL secret"), 500}} {
		service.err = tc.err
		for _, w := range []*httptest.ResponseRecorder{get(), mentorRequest(t, h.MentorProgress)} {
			if w.Code != tc.code || strings.Contains(w.Body.String(), "secret") {
				t.Fatal(w)
			}
		}
	}
}
