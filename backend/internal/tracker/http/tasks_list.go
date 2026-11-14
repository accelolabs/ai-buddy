package trackerhttp

import (
	"net/http"

	"ai-buddy/internal/platform/httpx"
)

func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	result, err := h.tasks.Tasks(r.Context(), httpx.Owner(r))
	if err != nil {
		Failure(w, err)
		return
	}

	httpx.JSON(w, 200, tasksResponse{Tasks: result})
}
