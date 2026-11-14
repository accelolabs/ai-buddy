package wikihttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-buddy/internal/users"
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

func TestHandlers(t *testing.T) {
	store := &store{}
	embedder := &embedder{}
	h := &Handler{Service: &wiki.Service{Store: store, Embedder: embedder, Minimum: 0.35}}
	call := func(handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/search", strings.NewReader(body))
		r.SetPathValue("id", "doc")
		r = r.WithContext(users.WithUser(r.Context(), users.User{ID: "intern", Team: "backend", Role: "intern"}))
		w := httptest.NewRecorder()
		handler(w, r)
		return w
	}
	for _, handler := range []http.HandlerFunc{h.List, h.Get, h.Search} {
		w := call(handler, `{"query":"text"}`)
		if w.Code != 200 || store.team != "backend" {
			t.Fatal(w, store.team)
		}
	}
	w := call(h.Get, "")
	var doc map[string]any
	if json.Unmarshal(w.Body.Bytes(), &doc) != nil || doc["markdown"] != "# Title" || doc["url"] != "/api/documents/doc" || len(doc) != 5 {
		t.Fatal(w)
	}
	for _, body := range []string{`{"query":" "}`, `{"query":"text","team":"platform"}`, `{} {}`, `{"query":"` + strings.Repeat("я", 4001) + `"}`} {
		if w = call(h.Search, body); w.Code != 400 {
			t.Fatal(w)
		}
	}
	store.empty = true
	if w = call(h.Search, `{"query":"text"}`); w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatal(w)
	}
	embedder.err = errors.New("model unavailable")
	if w = call(h.Search, `{"query":"text"}`); w.Code != 503 {
		t.Fatal(w)
	}
	embedder.err = nil
	store.err = errors.New("SQL secret")
	for _, handler := range []http.HandlerFunc{h.List, h.Get, h.Search} {
		w = call(handler, `{"query":"text"}`)
		if w.Code != 503 || strings.Contains(w.Body.String(), "secret") {
			t.Fatal(w)
		}
	}
}
