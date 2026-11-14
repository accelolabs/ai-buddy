package wikihttp

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
	"ai-buddy/internal/testutil"
	"ai-buddy/internal/wiki"
)

type store struct {
	err   error
	team  string
	empty bool
}

func (s *store) List(_ context.Context, team string) ([]wiki.Document, error) {
	s.team = team
	return []wiki.Document{{ID: "doc", Title: "Title", Path: "common/a.md"}}, s.err
}

func (s *store) Get(_ context.Context, team, id string) (wiki.Document, error) {
	s.team = team
	return wiki.Document{ID: id, Path: "common/a.md", Title: "Title", Markdown: "# Title"}, s.err
}

func (s *store) Search(_ context.Context, team, q string, v []float64, min float64, fp string) ([]wiki.Result, error) {
	s.team = team
	if s.empty {
		return []wiki.Result{}, s.err
	}
	return []wiki.Result{{DocumentID: "doc", DocumentTitle: "Title", SectionTitle: "Section", Anchor: "section", Text: "text"}}, s.err
}

type embedder struct{ err error }

func (e *embedder) Fingerprint(context.Context) (string, error) { return "fp", e.err }
func (e *embedder) Embed(context.Context, string) ([]float64, error) {
	return make([]float64, 768), e.err
}

type teams struct {
	err   error
	calls int
}

func (t *teams) Team(_ context.Context, id string) (string, error) {
	t.calls++
	return "platform", t.err
}

func TestRoutes(t *testing.T) {
	sessions := scs.New()
	login := auth.New(sessions, nil)
	s := &store{}
	e := &embedder{}
	users := &teams{}
	h := &Handler{&wiki.Service{Store: s, Embedder: e, Minimum: 0.35}, users}
	routes := testutil.Contract(t, "wiki", Routes(login, h, "buddy"))
	cookieFor := func(team string) *http.Cookie {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		sessions.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, _ := json.Marshal(auth.User{ID: "42", Role: "intern", Team: team})
			sessions.Put(r.Context(), auth.SessionKeyUser, string(raw))
		})).ServeHTTP(w, r)
		return w.Result().Cookies()[0]
	}
	cookie := cookieFor("backend")
	call := func(method, path, body, token string, session bool) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if session {
			r.AddCookie(cookie)
		}
		if token != "" {
			r.Header.Set("Authorization", token)
		}
		routes.ServeHTTP(w, r)
		return w
	}
	for _, test := range []struct {
		method, path, body, key string
		session                 bool
		code                    int
	}{
		{"GET", "/api/documents", "", "", false, 401},
		{"POST", "/api/search", `{"query":"text"}`, "Bearer buddy", false, 401},
		{"GET", "/api/documents", "", "", true, 200},
		{"GET", "/api/documents/doc", "", "", true, 200},
		{"POST", "/api/search", `{"query":"text"}`, "", true, 200},
		{"POST", "/api/search", `{"query":" "}`, "", true, 400},
		{"POST", "/api/search", `{"query":"text","team":"platform"}`, "", true, 400},
		{"POST", "/api/search", `{} {}`, "", true, 400},
		{"POST", "/internal/users/42/search", `{"query":"text"}`, "", true, 401},
		{"POST", "/internal/users/42/search", `{"query":"text"}`, "Bearer wrong", false, 401},
		{"POST", "/internal/users/42/search", `{"query":"text"}`, "Bearer buddy", false, 200},
		{"GET", "/internal/users/42/search", "", "Bearer buddy", false, 405},
		{"GET", "/api/search", "", "", true, 405},
		{"GET", "/other", "", "", false, 404},
	} {
		w := call(test.method, test.path, test.body, test.key, test.session)
		if w.Code != test.code {
			t.Fatalf("%s %s: %d %s", test.method, test.path, w.Code, w.Body.String())
		}
		if w.Header().Get("Content-Type") != "application/json" {
			t.Fatal(w.Header())
		}
	}
	if s.team != "platform" || users.calls != 1 {
		t.Fatal(s.team, users.calls)
	}
	// Rendering receives the source document; RAG section metadata stays in search results.
	detail := call("GET", "/api/documents/doc", "", "", true)
	var response map[string]any
	if json.Unmarshal(detail.Body.Bytes(), &response) != nil || len(response) != 5 || response["markdown"] != "# Title" || response["id"] != "doc" || response["path"] != "common/a.md" || response["title"] != "Title" || response["url"] != "/api/documents/doc" {
		t.Fatal("document contract", detail.Body.String())
	}
	for _, query := range []string{" ", strings.Repeat("я", 4001)} {
		body, _ := json.Marshal(map[string]string{"query": query})
		before := users.calls
		result := call("POST", "/internal/users/42/search", string(body), "Bearer buddy", false)
		if result.Code != 400 || users.calls != before {
			t.Fatal("invalid query reached Authentik", result.Code)
		}
	}
	// Browser reads continue to use their session snapshot, even if Authentik is down.
	users.err = wiki.ErrUnavailable
	if w := call("GET", "/api/documents", "", "", true); w.Code != 200 || s.team != "backend" {
		t.Fatal(w.Code, s.team)
	}
	for _, err := range []error{wiki.ErrUnavailable, wiki.ErrNotFound} {
		users.err = err
		w := call("POST", "/internal/users/42/search", `{"query":"text"}`, "Bearer buddy", false)
		want := 503
		if err == wiki.ErrNotFound {
			want = 404
		}
		if w.Code != want {
			t.Fatal(w.Code)
		}
	}
	for _, path := range []string{"/api/documents", "/api/documents/doc", "/api/search"} {
		s.err = errors.New("db")
		method, body := "GET", ""
		if path == "/api/search" {
			method = "POST"
			body = `{"query":"text"}`
		}
		if w := call(method, path, body, "", true); w.Code != 503 {
			t.Fatal(path, w.Code)
		}
	}
	s.err = wiki.ErrNotFound
	if w := call("GET", "/api/documents/foreign", "", "", true); w.Code != 404 {
		t.Fatal(w.Code)
	}
	s.err = nil
	s.empty = true
	if w := call("POST", "/api/search", `{"query":"absent"}`, "", true); w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatal(w.Code, w.Body)
	}
	e.err = wiki.ErrUnavailable
	if w := call("POST", "/api/search", `{"query":"text"}`, "", true); w.Code != 503 {
		t.Fatal(w.Code)
	}
	cookie = cookieFor("platform")
	if w := call("GET", "/api/documents", "", "", true); w.Code != 200 || s.team != "platform" {
		t.Fatal(w.Code, s.team)
	}
}
