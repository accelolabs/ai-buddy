package buddyhttp

import (
	"net/http"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/platform/httpx"
)

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	var in confirmRequest
	if httpx.Decode(w, r, &in) != nil {
		fail(w, buddy.ErrInvalid)
		return
	}

	a, e := h.Service.Confirm(r.Context(), owner(r), r.PathValue("id"), in.Fields)
	if e != nil {
		fail(w, e)
		return
	}
	httpx.JSON(w, 200, action(a))
}
func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	a, e := h.Service.Cancel(r.Context(), owner(r), r.PathValue("id"))
	if e != nil {
		fail(w, e)
		return
	}
	httpx.JSON(w, 200, action(a))
}
