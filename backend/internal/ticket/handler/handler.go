package handler

import (
	"backend/internal/ticket"
	"backend/internal/users"
	"backend/internal/users/middleware"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

type ticketRepository interface {
	List(context.Context, users.User) ([]ticket.Ticket, error)
	Get(context.Context, users.User, string) (ticket.Ticket, error)
}

type Handler struct{ repository ticketRepository }

type ticketResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewHandler(repository ticketRepository) *Handler { return &Handler{repository: repository} }

func ticketView(value ticket.Ticket) ticketResponse {
	return ticketResponse{ID: value.ID, UserID: value.UserID, Title: value.Title, Description: value.Description, CreatedAt: value.CreatedAt}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "select a user", http.StatusUnauthorized)
		return
	}
	values, err := h.repository.List(r.Context(), user)
	if err != nil {
		log.Printf("list tickets: %v", err)
		http.Error(w, "failed to list tickets", http.StatusInternalServerError)
		return
	}
	response := make([]ticketResponse, 0, len(values))
	for _, value := range values {
		response = append(response, ticketView(value))
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write ticket list: %v", err)
	}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "select a user", http.StatusUnauthorized)
		return
	}
	value, err := h.repository.Get(r.Context(), user, r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("get ticket: %v", err)
		http.Error(w, "failed to get ticket", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ticketView(value)); err != nil {
		log.Printf("write ticket: %v", err)
	}
}
