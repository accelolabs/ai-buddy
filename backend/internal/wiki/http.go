package wiki

import (
	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/users"
	"context"
	"errors"
	"net/http"
	"time"
)

func (s *Service) ListHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	u := users.CurrentUser(ctx)

	docs, err := s.Store.List(ctx, u.Team)
	if err != nil {
		fail(w, err)
		return
	}

	out := []metadata{}

	for _, d := range docs {
		out = append(out, documentMetadata(d))
	}

	httpx.JSON(w, 200, out)
}

func (s *Service) GetHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	u := users.CurrentUser(ctx)

	d, err := s.Store.Get(ctx, u.Team, r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}

	out := document{
		metadata: documentMetadata(d),
		Markdown: d.Markdown,
	}

	httpx.JSON(w, 200, out)
}

type metadata struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

type document struct {
	metadata
	Markdown string `json:"markdown"`
}

func documentMetadata(d Document) metadata {
	return metadata{
		ID:    d.ID,
		Path:  d.Path,
		Title: d.Title,
		URL:   "/api/documents/" + d.ID,
	}
}

func fail(w http.ResponseWriter, err error) {
	code := 503
	message := "service unavailable"

	if errors.Is(err, ErrNotFound) {
		code = 404
		message = "not found"
	}

	httpx.JSON(w, code, map[string]string{"error": message})
}

func Register(mux *http.ServeMux, login *users.Selector, s *Service) {
	mux.Handle("GET /api/documents", login.RequireUser(http.HandlerFunc(s.ListHTTP)))
	mux.Handle("GET /api/documents/{id}", login.RequireUser(http.HandlerFunc(s.GetHTTP)))
}
