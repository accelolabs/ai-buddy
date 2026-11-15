package wiki

import (
	"backend-ai/internal/users"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type store struct {
	err  error
	team string
}

func (s *store) List(_ context.Context, team string) ([]Document, error) {
	s.team = team
	return []Document{{ID: "doc", Title: "Title", Path: "common/a.md"}}, s.err
}

func (s *store) Get(_ context.Context, team, id string) (Document, error) {
	s.team = team
	return Document{ID: id, Path: "common/a.md", Title: "Title", Markdown: "# Title"}, s.err
}

func (*store) Search(
	context.Context,
	string,
	string,
	[]float64,
	float64,
	string,
) ([]Source, error) {
	panic("document HTTP handlers must not invoke search")
}

func TestHandlers(t *testing.T) {
	store := &store{}
	h := &Service{Store: store}

	call := func(handler http.HandlerFunc) *httptest.ResponseRecorder {
		r := httptest.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"/api/documents/doc",
			nil,
		)
		r.SetPathValue("id", "doc")
		r = r.WithContext(
			users.WithUser(
				r.Context(),
				users.User{ID: "intern", Team: "backend", Role: "intern"},
			),
		)
		w := httptest.NewRecorder()
		handler(w, r)

		return w
	}
	for _, handler := range []http.HandlerFunc{h.ListHTTP, h.GetHTTP} {
		w := call(handler)
		if w.Code != 200 || store.team != "backend" {
			t.Fatal(w, store.team)
		}
	}

	w := call(h.GetHTTP)

	var doc map[string]any
	if json.Unmarshal(w.Body.Bytes(), &doc) != nil || doc["markdown"] != "# Title" ||
		doc["url"] != "/api/documents/doc" ||
		len(doc) != 5 {
		t.Fatal(w)
	}

	store.err = errors.New("SQL secret")

	for _, handler := range []http.HandlerFunc{h.ListHTTP, h.GetHTTP} {
		w = call(handler)
		if w.Code != 503 || strings.Contains(w.Body.String(), "secret") {
			t.Fatal(w)
		}
	}
}
