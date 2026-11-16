package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	// Project packages.
	"backend/internal/users"
	"backend/internal/users/middleware"
)

type userLister interface {
	List(context.Context) ([]users.User, error)
	ListInterns(context.Context, string) ([]users.User, error)
}

type Handler struct {
	repository userLister
}

type userResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
	Team string `json:"team"`
}

func NewHandler(repository userLister) *Handler {
	return &Handler{
		repository: repository,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.repository.List(r.Context())
	if err != nil {
		log.Printf("list users: %v", err)
		http.Error(w, "failed to list users", http.StatusInternalServerError)

		return
	}

	response := make([]userResponse, len(users))

	for i, user := range users {
		response[i] = userResponse(user)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write users response: %v", err)
	}
}

func (h *Handler) ListInterns(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "select a user", http.StatusUnauthorized)

		return
	}

	if user.Role != users.RoleMentor {
		http.Error(w, "only mentors can list interns", http.StatusForbidden)

		return
	}

	values, err := h.repository.ListInterns(r.Context(), user.ID)
	if err != nil {
		log.Printf("list interns: %v", err)
		http.Error(w, "failed to list interns", http.StatusInternalServerError)

		return
	}

	response := make([]userResponse, len(values))

	for i, value := range values {
		response[i] = userResponse(value)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write interns: %v", err)
	}
}
