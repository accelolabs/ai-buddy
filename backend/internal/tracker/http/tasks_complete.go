package trackerhttp

import (
	"net/http"

	"ai-buddy/internal/platform/httpx"
)

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	result, err := h.tasks.Complete(r.Context(), httpx.Owner(r), r.PathValue("taskID"))
	if err != nil {
		Failure(w, err)
		return
	}

	httpx.JSON(w, 200, result)
}
