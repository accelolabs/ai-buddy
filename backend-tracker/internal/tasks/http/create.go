package taskshttp

import (
	"net/http"

	"ai-buddy/backend-tracker/internal/httpx"
	"ai-buddy/backend-tracker/internal/tasks"
)

type createRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Deadline    string `json:"deadline"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request createRequest
	if err := httpx.Decode(w, r, &request); err != nil {
		httpx.Failure(w, err)
		return
	}

	input := tasks.CreateInput{
		Title:       request.Title,
		Description: request.Description,
		Deadline:    request.Deadline,
	}
	result, created, err := h.service.Create(r.Context(), httpx.Owner(r), r.Header.Get("Idempotency-Key"), input)
	if err != nil {
		httpx.Failure(w, err)
		return
	}

	code := 200
	if created {
		code = 201
	}

	httpx.JSON(w, code, fromTask(result))
}
