package trackerhttp

import (
	"net/http"

	"ai-buddy/internal/platform/httpx"
)

func (h *Handler) Progress(w http.ResponseWriter, r *http.Request) {
	result, err := h.progress.Progress(r.Context(), httpx.Owner(r))
	if err != nil {
		Failure(w, err)
		return
	}

	httpx.JSON(w, 200, result)
}
