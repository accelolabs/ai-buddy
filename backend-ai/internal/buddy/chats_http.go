package buddy

import (
	"backend-ai/internal/platform/httpx"
	"backend-ai/internal/users"
	"net/http"
)

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	c, e := h.Service.Repo.CreateChat(r.Context(), users.CurrentUser(r.Context()).ID)
	if e != nil {
		fail(w, e)
		return
	}

	httpx.JSON(w, 201, c)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	all, e := h.Service.Repo.Chats(r.Context(), users.CurrentUser(r.Context()).ID)
	if e != nil {
		fail(w, e)
		return
	}

	httpx.JSON(w, 200, map[string]any{"chats": all})
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.Repo.History(
		r.Context(),
		users.CurrentUser(r.Context()).ID,
		r.PathValue("id"),
	)
	if e != nil {
		fail(w, e)
		return
	}

	runs := []runDTO{}

	for _, v := range v.Runs {
		runs = append(runs, run(v))
	}

	var latest *runDTO

	if len(runs) > 0 {
		latest = &runs[len(runs)-1]
	}

	httpx.JSON(
		w,
		200,
		map[string]any{
			"chat":       v.Chat,
			"messages":   v.Messages,
			"runs":       runs,
			"latest_run": latest,
			"actions":    v.Actions,
			"cards":      v.Cards,
		},
	)
}
