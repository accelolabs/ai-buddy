package handler

import (
	"backend/internal/buddy"
	"backend/internal/buddy/dialog"
	"backend/internal/users"
	"backend/internal/users/middleware"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type ticketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type recipientResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ticketResponse struct {
	Status      string            `json:"status"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Recipient   recipientResponse `json:"recipient"`
	TicketID    *string           `json:"ticket_id"`
}

func ticketView(card *buddy.Ticket) *ticketResponse {
	if card == nil {
		return nil
	}
	status := "draft"
	if card.TicketID != nil {
		status = "created"
	}
	return &ticketResponse{Status: status, Title: card.Title, Description: card.Description, Recipient: recipientResponse(card.Recipient), TicketID: card.TicketID}
}

func (h *Handler) ConfirmTicket(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "select a user", http.StatusUnauthorized)
		return
	}
	if user.Role != users.RoleIntern {
		http.Error(w, "only interns can confirm tickets", http.StatusForbidden)
		return
	}
	messageID, err := strconv.ParseInt(r.PathValue("messageID"), 10, 64)
	if err != nil || messageID <= 0 {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}
	var input ticketRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid ticket", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		http.Error(w, "invalid ticket", http.StatusBadRequest)
		return
	}
	input.Title, input.Description = strings.TrimSpace(input.Title), strings.TrimSpace(input.Description)
	if input.Title == "" || input.Description == "" {
		http.Error(w, "invalid ticket", http.StatusBadRequest)
		return
	}
	card, created, err := h.dialog.ConfirmTicket(r.Context(), user, r.PathValue("chatID"), messageID, input.Title, input.Description)
	switch {
	case errors.Is(err, dialog.ErrForbidden):
		http.Error(w, "only interns can confirm tickets", http.StatusForbidden)
	case errors.Is(err, dialog.ErrInvalidTicket):
		http.Error(w, "invalid ticket", http.StatusBadRequest)
	case errors.Is(err, dialog.ErrNotFound):
		http.NotFound(w, r)
	case errors.Is(err, buddy.ErrMentorChanged):
		http.Error(w, "assigned mentor changed", http.StatusConflict)
	case err != nil:
		log.Printf("confirm ticket: %v", err)
		http.Error(w, "failed to confirm ticket", http.StatusInternalServerError)
	default:
		w.Header().Set("Content-Type", "application/json")
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(ticketView(&card)); err != nil {
			log.Printf("write ticket card: %v", err)
		}
	}
}
