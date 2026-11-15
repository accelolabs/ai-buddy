package buddy

import (
	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/users"
	"net/http"
)

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	var in confirmRequest
	if httpx.Decode(w, r, &in) != nil {
		fail(w, ErrInvalid)
		return
	}

	a, e := h.Service.Confirm(
		r.Context(),
		users.CurrentUser(r.Context()),
		r.PathValue("id"),
		in.Fields,
	)
	if e != nil {
		fail(w, e)
		return
	}

	httpx.JSON(w, 200, a)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	a, e := h.Service.Cancel(r.Context(), users.CurrentUser(r.Context()), r.PathValue("id"))
	if e != nil {
		fail(w, e)
		return
	}

	httpx.JSON(w, 200, a)
}
