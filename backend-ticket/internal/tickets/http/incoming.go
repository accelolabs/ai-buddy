package ticketshttp

import (
	"net/http"

	"ai-buddy/backend-ticket/internal/httpx"
	"ai-buddy/backend-ticket/internal/tickets"
)

func (h *Handler) Incoming(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.Incoming(r.Context(), httpx.Owner(r))
	if err != nil {
		httpx.Failure(w, err)
		return
	}
	if list == nil {
		list = []tickets.Ticket{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"tickets": list})
}
