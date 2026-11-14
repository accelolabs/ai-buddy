package ticketshttp

import (
	"net/http"

	"ai-buddy/internal/platform/httpx"
	domainhttp "ai-buddy/internal/ticket/httpx"
	"ai-buddy/internal/ticket/tickets"
)

type createRequest struct {
	Question    string `json:"question"`
	Description string `json:"description"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input createRequest
	if err := httpx.Decode(w, r, &input); err != nil {
		domainhttp.Failure(w, err)
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
		domainhttp.Failure(w, err)
		return
	}
	code := http.StatusOK
	if created {
		code = http.StatusCreated
	}
	httpx.JSON(w, code, fromTicket(ticket))
}
