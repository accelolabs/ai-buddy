package wikihttp

import (
	"context"
	"net/http"
	"time"

	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/users"
)

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	u, _ := users.CurrentUser(ctx)
	docs, err := h.Service.Store.List(ctx, u.Team)
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

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	u, _ := users.CurrentUser(ctx)
	d, err := h.Service.Store.Get(ctx, u.Team, r.PathValue("id"))
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
