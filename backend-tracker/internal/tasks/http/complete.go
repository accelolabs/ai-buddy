package taskshttp

import (
	"net/http"

	"ai-buddy/backend-tracker/internal/httpx"
)

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Complete(r.Context(), httpx.Owner(r), r.PathValue("taskID"))
	if err != nil {
		httpx.Failure(w, err)
		return
	}
	httpx.JSON(w, 200, result)
}
