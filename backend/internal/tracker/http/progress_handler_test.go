package trackerhttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"

	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/tracker/apperror"
	"ai-buddy/internal/tracker/interns"
	"ai-buddy/internal/tracker/progress"
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

func TestProgressHandlers(t *testing.T) {
	service := &progressStub{}
	h := New(nil, nil, service)
	get := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/internal/interns/i/progress", nil)
		r.SetPathValue("id", "i")
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
