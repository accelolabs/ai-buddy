package handler

import (
	"backend/internal/ticket"
	"backend/internal/users"
	"backend/internal/users/middleware"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type ticketRepository interface {
	Add(context.Context, ticket.Ticket) (ticket.Ticket, error)
	List(context.Context, users.User) ([]ticket.Ticket, error)
	Get(context.Context, users.User, string) (ticket.Ticket, error)
}

type Handler struct{ repository ticketRepository }

type ticketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "select a user", http.StatusUnauthorized)
		return
	}
	if user.Role != users.RoleIntern {
		http.Error(w, "only interns can create tickets", http.StatusForbidden)
		return
	}
	var input ticketRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid ticket", http.StatusBadRequest)
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if err := decoder.Decode(new(any)); err != io.EOF || input.Title == "" || input.Description == "" {
		http.Error(w, "invalid ticket", http.StatusBadRequest)
		return
	}
	value, err := h.repository.Add(r.Context(), ticket.Ticket{ID: rand.Text(), UserID: user.ID, Title: input.Title, Description: input.Description})
	if err != nil {
		log.Printf("create ticket: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(ticketView(value)); err != nil {
		log.Printf("write ticket: %v", err)
	}
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
