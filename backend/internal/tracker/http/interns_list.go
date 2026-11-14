package trackerhttp

import (
	"net/http"

	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/users"
)

func (h *Handler) ListInterns(w http.ResponseWriter, r *http.Request) {
	u, _ := users.CurrentUser(r.Context())
	result, err := h.profiles.Interns(r.Context(), u.ID)
	if err != nil {
		Failure(w, err)
		return
	}

	httpx.JSON(w, 200, internsResponse{Interns: result})
}
