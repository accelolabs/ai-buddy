package trackerhttp

import (
	"net/http"

	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/users"
)

func (h *Handler) MentorProgress(w http.ResponseWriter, r *http.Request) {
	u, _ := users.CurrentUser(r.Context())
	result, err := h.progress.MentorProgress(r.Context(), u.ID, r.PathValue("id"))
	if err != nil {
		Failure(w, err)
		return
	}

	httpx.JSON(w, 200, result)
}
