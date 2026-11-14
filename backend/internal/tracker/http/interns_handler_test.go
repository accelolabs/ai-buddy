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
	"ai-buddy/internal/tracker/interns"
)

type internsStub struct {
	err    error
	mentor *interns.Profile
	id     string
}

func (s *internsStub) Interns(_ context.Context, id string) ([]interns.Profile, error) {
	s.id = id
	return []interns.Profile{{ID: "i"}}, s.err
}

func (s *internsStub) Mentor(_ context.Context, id string) (*interns.Profile, error) {
	s.id = id
	return s.mentor, s.err
}

func sessionRequest(t *testing.T, handler http.HandlerFunc) *httptest.ResponseRecorder {
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

	r := httptest.NewRequest("GET", "/api/interns", nil)
	r.AddCookie(&http.Cookie{Name: sessions.Cookie.Name, Value: token})
	r.SetPathValue("id", "i")
	a := auth.New(sessions, nil)
	w := httptest.NewRecorder()
	a.WithSession(a.RequireRoles(handler, "mentor")).ServeHTTP(w, r)
	return w
}

func TestInternsHandlers(t *testing.T) {
	service := &internsStub{mentor: &interns.Profile{ID: "mentor"}}
	h := New(nil, service, nil)
	w := sessionRequest(t, h.ListInterns)
	if w.Code != 200 || service.id != "m" || !strings.Contains(w.Body.String(), `"interns":[`) {
		t.Fatal(w, service)
	}

	mentorRequest := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/internal/interns/i/mentor", nil)
		r.SetPathValue("id", "i")
		w := httptest.NewRecorder()
		h.Mentor(w, r)
		return w
	}
	w = mentorRequest()
	if w.Code != 200 || service.id != "i" || !strings.Contains(w.Body.String(), `"authentik_id":"mentor"`) {
		t.Fatal(w, service)
	}

	service.mentor = nil
	w = mentorRequest()
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"mentor":null`) {
		t.Fatal(w)
	}

	service.err = errors.New("SQL secret")
	for _, w := range []*httptest.ResponseRecorder{sessionRequest(t, h.ListInterns), mentorRequest()} {
		if w.Code != 500 || strings.Contains(w.Body.String(), "secret") {
			t.Fatal(w)
		}
	}
}

func (s *internsStub) Intern(_ context.Context, id string) (interns.Profile, error) {
	return interns.Profile{ID: id}, s.err
}
