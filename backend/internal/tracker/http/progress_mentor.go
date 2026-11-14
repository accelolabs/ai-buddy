package trackerhttp

import (
	"net/http"

	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/platform/httpx"
)

func (h *Handler) MentorProgress(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.CurrentUser(r.Context())
	result, err := h.progress.MentorProgress(r.Context(), u.ID, r.PathValue("id"))
	if err != nil {
		Failure(w, err)
		return
	}

	httpx.JSON(w, 200, result)
}
