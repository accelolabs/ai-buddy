package buddyhttp

import (
	"net/http"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/platform/httpx"
)

func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	var in sendRequest
	if httpx.Decode(w, r, &in) != nil {
		fail(w, buddy.ErrInvalid)
		return
	}
	v, e := h.Service.Send(r.Context(), owner(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), in.Content)
	if e != nil {
		fail(w, e)
		return
	}
	httpx.JSON(w, 202, run(v))
}
func (h *Handler) retry(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.Retry(r.Context(), owner(r), r.PathValue("id"))
	if e != nil {
		fail(w, e)
		return
	}
	httpx.JSON(w, 202, run(v))
}
