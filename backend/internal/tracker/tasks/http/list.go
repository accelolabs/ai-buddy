package taskshttp

import (
	"ai-buddy/internal/tracker/dto"
	"net/http"

	"ai-buddy/internal/platform/httpx"
	domainhttp "ai-buddy/internal/tracker/httpx"
)

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Tasks(r.Context(), httpx.Owner(r))
	if err != nil {
		domainhttp.Failure(w, err)
		return
	}

	httpx.JSON(w, 200, tasksResponse{Tasks: dto.Tasks(result)})
}
