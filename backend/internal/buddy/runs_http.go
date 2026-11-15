package buddy

import (
	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/users"
	"net/http"
)

func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	var in sendRequest
	if httpx.Decode(w, r, &in) != nil {
		fail(w, ErrInvalid)
		return
	}

	v, e := h.Service.Send(
		r.Context(),
		users.CurrentUser(r.Context()),
		r.PathValue("id"),
		r.Header.Get("Idempotency-Key"),
		in.Content,
	)
	if e != nil {
		fail(w, e)
		return
	}

	httpx.JSON(w, 202, run(v))
}

func (h *Handler) retry(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.Retry(r.Context(), users.CurrentUser(r.Context()), r.PathValue("id"))
	if e != nil {
		fail(w, e)
		return
	}

	httpx.JSON(w, 202, run(v))
}
