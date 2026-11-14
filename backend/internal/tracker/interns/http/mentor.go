package internshttp

import (
	"net/http"

	"ai-buddy/internal/tracker/httpx"
)

func (h *Handler) Mentor(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Mentor(r.Context(), httpx.Owner(r))
	if err != nil {
		httpx.Failure(w, err)
		return
	}

	httpx.JSON(w, 200, fromMentor(result))
}
