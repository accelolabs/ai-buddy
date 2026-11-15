package handler

import (
	"backend/internal/users"
	"context"
	"encoding/json"
	"log"
	"net/http"
)

type userLister interface {
	List(context.Context) ([]users.User, error)
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
	return &Handler{repository: repository}
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
