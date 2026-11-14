package ticketshttp

import (
	"net/http"

	"ai-buddy/backend-ticket/internal/httpx"
	"ai-buddy/backend-ticket/internal/tickets"
)

type createRequest struct {
	Question    string `json:"question"`
	Description string `json:"description"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input createRequest
	if err := httpx.Decode(w, r, &input); err != nil {
		httpx.Failure(w, err)
		return
	}
	ticket, created, err := h.service.Create(
		r.Context(),
		httpx.Owner(r),
		r.Header.Get("Idempotency-Key"),
		tickets.CreateInput{
			Question:    input.Question,
			Description: input.Description,
		},
	)
	if err != nil {
		httpx.Failure(w, err)
		return
	}
	code := http.StatusOK
	if created {
		code = http.StatusCreated
	}
	httpx.JSON(w, code, ticket)
}
