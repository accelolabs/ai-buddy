package ticketshttp

import (
	"net/http"

	"ai-buddy/internal/platform/httpx"
	domainhttp "ai-buddy/internal/ticket/httpx"
)

func (h *Handler) Sent(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.Sent(r.Context(), httpx.Owner(r))
	if err != nil {
		domainhttp.Failure(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, fromTickets(list))
}
