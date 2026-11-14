package progresshttp

import (
	"net/http"

	"ai-buddy/backend-tracker/internal/auth"
	"ai-buddy/backend-tracker/internal/httpx"
)

func (h *Handler) Mentor(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.CurrentUser(r.Context())
	result, err := h.service.MentorProgress(r.Context(), u.ID, r.PathValue("id"))
	if err != nil {
		httpx.Failure(w, err)
		return
	}

	httpx.JSON(w, 200, result)
}
