package progresshttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-buddy/backend-tracker/internal/apperror"
	"ai-buddy/backend-tracker/internal/auth"
	"ai-buddy/backend-tracker/internal/interns"
	"ai-buddy/backend-tracker/internal/progress"
	"github.com/alexedwards/scs/v2"
)

type serviceStub struct {
	err          error
	id, mentorID string
}

func (s *serviceStub) Progress(_ context.Context, id string) (progress.Progress, error) {
	s.id = id
	return progress.Progress{Intern: interns.Profile{ID: id}}, s.err
}

func (s *serviceStub) MentorProgress(_ context.Context, mentor, id string) (progress.Progress, error) {
	s.id = id
	s.mentorID = mentor
	return progress.Progress{Intern: interns.Profile{ID: id}}, s.err
}

func mentorRequest(t *testing.T, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	sessions := scs.New()
	ctx, err := sessions.Load(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}

	raw, _ := json.Marshal(auth.User{
		ID:   "m",
		Role: "mentor",
		Team: "backend",
	})
	sessions.Put(ctx, auth.SessionKeyUser, string(raw))
	token, _, err := sessions.Commit(ctx)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/api/interns/i/progress", nil)
	r.SetPathValue("id", "i")
	r.AddCookie(&http.Cookie{Name: sessions.Cookie.Name, Value: token})
	a := auth.New(sessions, nil)
	w := httptest.NewRecorder()
	a.WithSession(a.RequireRoles(handler, "mentor")).ServeHTTP(w, r)
	return w
}

func TestHandlers(t *testing.T) {
	service := &serviceStub{}
	h := New(service)
	get := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/internal/interns/i/progress", nil)
		r.SetPathValue("id", "i")
		w := httptest.NewRecorder()
		h.Get(w, r)
		return w
	}
	w := get()
	if w.Code != 200 || service.id != "i" {
		t.Fatal(w, service)
	}

	w = mentorRequest(t, h.Mentor)
	if w.Code != 200 || service.id != "i" || service.mentorID != "m" {
		t.Fatal(w, service)
	}

	for _, tc := range []struct {
		err  error
		code int
	}{{apperror.ErrNotFound, 404}, {errors.New("SQL secret"), 500}} {
		service.err = tc.err
		for _, w := range []*httptest.ResponseRecorder{get(), mentorRequest(t, h.Mentor)} {
			if w.Code != tc.code || strings.Contains(w.Body.String(), "secret") {
				t.Fatal(w)
			}
		}
	}
}
