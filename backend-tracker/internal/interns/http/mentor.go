package internshttp

import (
	"net/http"

	"ai-buddy/backend-tracker/internal/httpx"
)

func (h *Handler) Mentor(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Mentor(r.Context(), httpx.Owner(r))
	if err != nil {
		httpx.Failure(w, err)
		return
	}
	httpx.JSON(w, 200, map[string]any{"mentor": result})
}
