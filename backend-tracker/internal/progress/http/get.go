package progresshttp

import (
	"net/http"

	"ai-buddy/backend-tracker/internal/httpx"
)

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Progress(r.Context(), httpx.Owner(r))
	if err != nil {
		httpx.Failure(w, err)
		return
	}
	httpx.JSON(w, 200, result)
}
