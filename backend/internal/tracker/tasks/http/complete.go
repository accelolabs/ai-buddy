package taskshttp

import (
	"ai-buddy/internal/tracker/dto"
	"net/http"

	"ai-buddy/internal/platform/httpx"
	domainhttp "ai-buddy/internal/tracker/httpx"
)

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Complete(r.Context(), httpx.Owner(r), r.PathValue("taskID"))
	if err != nil {
		domainhttp.Failure(w, err)
		return
	}

	httpx.JSON(w, 200, dto.FromTask(result))
}
