package trackerhttp

import (
	"net/http"

	"ai-buddy/internal/platform/httpx"
)

func (h *Handler) Mentor(w http.ResponseWriter, r *http.Request) {
	if _, err := h.profiles.Intern(r.Context(), httpx.Owner(r)); err != nil {
		Failure(w, err)
		return
	}
	result, err := h.profiles.Mentor(r.Context(), httpx.Owner(r))
	if err != nil {
		Failure(w, err)
		return
	}

	httpx.JSON(w, 200, mentorResponse{Mentor: result})
}
