package wikihttp

import (
	"context"
	"net/http"
	"strings"
	"time"

	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/wiki"
)

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	var body struct {
		Query string `json:"query"`
	}
	if httpx.Decode(w, r, &body) != nil {
		httpx.JSON(w, 400, map[string]string{"error": "invalid input"})
		return
	}

	body.Query = strings.TrimSpace(body.Query)
	if body.Query == "" || len([]rune(body.Query)) > 4000 {
		fail(w, wiki.ErrQuery)
		return
	}
	u, _ := auth.CurrentUser(ctx)
	team := u.Team
	if r.PathValue("id") != "" {
		var err error
		team, err = h.Teams.Team(ctx, r.PathValue("id"))
		if err != nil {
			fail(w, err)
			return
		}
	}

	found, err := h.Service.Search(ctx, team, body.Query)
	if err != nil {
		fail(w, err)
		return
	}

	out := []result{}
	for _, v := range found {
		out = append(out, result{
			DocumentID:    v.DocumentID,
			DocumentTitle: v.DocumentTitle,
			SectionTitle:  v.SectionTitle,
			Anchor:        v.Anchor,
			Text:          v.Text,
			URL:           "/api/documents/" + v.DocumentID,
		})
	}
	httpx.JSON(w, 200, out)
}
