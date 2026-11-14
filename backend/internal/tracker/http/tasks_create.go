package trackerhttp

import (
	"net/http"

	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/tracker/tasks"
)

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input tasks.CreateInput
	if err := httpx.Decode(w, r, &input); err != nil {
		Failure(w, err)
		return
	}

	result, created, err := h.tasks.Create(r.Context(), httpx.Owner(r), r.Header.Get("Idempotency-Key"), input)
	if err != nil {
		Failure(w, err)
		return
	}

	code := 200
	if created {
		code = 201
	}

	httpx.JSON(w, code, result)
}
