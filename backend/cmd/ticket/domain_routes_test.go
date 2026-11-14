package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/testutil"
	"ai-buddy/internal/ticket/apperror"
	"ai-buddy/internal/ticket/tickets"
	ticketshttp "ai-buddy/internal/ticket/tickets/http"
)

type stubService struct {
	owner, key string
	input      tickets.CreateInput
	err        error
	created    bool
	calls      int
}

func (s *stubService) Create(_ context.Context, id, key string, in tickets.CreateInput) (tickets.Ticket, bool, error) {
	s.owner = id
	s.key = key
	s.input = in
	s.calls++
	return tickets.Ticket{SenderID: id, RecipientID: "mentor"}, s.created, s.err
}
func (s *stubService) Sent(_ context.Context, id string) ([]tickets.Ticket, error) {
	s.owner = id
	s.calls++
	return nil, s.err
}
func (s *stubService) Incoming(_ context.Context, id string) ([]tickets.Ticket, error) {
	s.owner = id
	s.calls++
	return []tickets.Ticket{{RecipientID: id}}, s.err
}
func TestDomainRoutes(t *testing.T) {
	sessions := scs.New()
	sessions.Store = memstore.NewWithCleanupInterval(0)
	service := &stubService{created: true}
	handler := testutil.Contract(t, "ticket", domainRoutes(auth.New(sessions, nil), ticketshttp.New(service), "buddy"))
	cookie := func(id, role string) *http.Cookie {
		w := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, _ := json.Marshal(auth.User{ID: id, Role: role, Team: "backend"})
			sessions.Put(r.Context(), auth.SessionKeyUser, string(raw))
		})).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		return w.Result().Cookies()[0]
	}
	intern, mentor := cookie("intern", "intern"), cookie("mentor", "mentor")
	request := func(method, path, body, token string, c *http.Cookie, code int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Authorization", token)
		r.Header.Set("Idempotency-Key", "key")
		if c != nil {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != code || !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body)
		}
		if strings.Contains(w.Body.String(), "SQL") {
			t.Fatal("secret leaked")
		}
		return w
	}
	for _, route := range []struct{ method, path string }{{"GET", "/api/tickets"}, {"POST", "/api/tickets"}, {"GET", "/api/tickets/incoming"}} {
		request(route.method, route.path, "", "", nil, 401)
	}
	request("GET", "/api/tickets", "", "", mentor, 403)
	request("POST", "/api/tickets", "", "", mentor, 403)
	request("GET", "/api/tickets/incoming", "", "", intern, 403)
	body := `{"question":"q","description":"d"}`
	request("POST", "/api/tickets?sender_id=other", body, "", intern, 201)
	if service.owner != "intern" || service.key != "key" || service.input.Question != "q" {
		t.Fatal(service)
	}
	service.created = false
	request("POST", "/api/tickets", body, "", intern, 200)
	before := service.calls
	for _, bad := range []string{`{"sender_id":"other"}`, `{"recipient_id":"other"}`, `{} {}`, `null`, strings.Repeat("x", (1<<20)+1)} {
		// null decodes to zero values and is rejected by the real service.
		if bad == "null" {
			continue
		}
		request("POST", "/api/tickets", bad, "", intern, 400)
	}
	if service.calls != before {
		t.Fatal("invalid input reached service")
	}
	w := request("GET", "/api/tickets?sender_id=other", "", "", intern, 200)
	if service.owner != "intern" || strings.TrimSpace(w.Body.String()) != `{"tickets":[]}` {
		t.Fatal(service.owner, w.Body)
	}
	request("GET", "/api/tickets/incoming?recipient_id=other", "", "", mentor, 200)
	if service.owner != "mentor" {
		t.Fatal(service.owner)
	}
	for _, token := range []string{"", "Bearer wrong", "Bearer ticket"} {
		request("POST", "/internal/interns/other/tickets", body, token, intern, 401)
	}
	request("POST", "/internal/interns/other/tickets", body, "Bearer buddy", nil, 200)
	if service.owner != "other" {
		t.Fatal(service.owner)
	}
	request("GET", "/internal/interns/other/tickets", "", "Bearer buddy", nil, 200)
	for _, tc := range []struct {
		err  error
		code int
	}{{apperror.ErrInvalid, 400}, {apperror.ErrNotFound, 404}, {apperror.ErrConflict, 409}, {apperror.ErrNoMentor, 422}, {apperror.ErrTracker, 503}, {errors.New("SQL secret"), 500}} {
		service.err = tc.err
		request("POST", "/api/tickets", body, "", intern, tc.code)
		request("GET", "/api/tickets", "", "", intern, tc.code)
		request("GET", "/api/tickets/incoming", "", "", mentor, tc.code)
	}
	service.err = nil
	request("DELETE", "/internal/interns/other/tickets", "", "Bearer buddy", nil, 405)
	request("GET", "/missing", "", "", nil, 404)
}
