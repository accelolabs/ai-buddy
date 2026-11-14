package internshttp

import (
	"net/http"

	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/tracker/httpx"
)

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.CurrentUser(r.Context())
	result, err := h.service.Interns(r.Context(), u.ID)
	if err != nil {
		httpx.Failure(w, err)
		return
	}

	httpx.JSON(w, 200, internsResponse{Interns: fromProfiles(result)})
}
