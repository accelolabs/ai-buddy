package progresshttp

import (
	"net/http"

	"ai-buddy/internal/tracker/httpx"
)

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Progress(r.Context(), httpx.Owner(r))
	if err != nil {
		httpx.Failure(w, err)
		return
	}

	httpx.JSON(w, 200, fromProgress(result))
}
