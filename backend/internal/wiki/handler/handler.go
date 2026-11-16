package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	// Project packages.
	"backend/internal/users"
	"backend/internal/users/middleware"
	"backend/internal/wiki"
	"backend/internal/wiki/service"
)

type documentService interface {
	List(context.Context, users.User) ([]wiki.Document, error)
	Get(context.Context, users.User, string) ([]byte, error)
}

type Handler struct {
	service documentService
}

type documentResponse struct {
	ID    string `json:"id"`
	Team  string `json:"team"`
	Title string `json:"title"`
}

func NewHandler(service documentService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "select a user", http.StatusUnauthorized)

		return
	}

	documents, err := h.service.List(r.Context(), user)
	if err != nil {
		log.Printf("list wiki: %v", err)
		http.Error(w, "failed to list articles", http.StatusInternalServerError)

		return
	}

	response := make([]documentResponse, 0, len(documents))

	for _, document := range documents {
		response = append(response, documentResponse{
			ID:    document.ID,
			Team:  document.Team,
			Title: document.Title,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write wiki list: %v", err)
	}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "select a user", http.StatusUnauthorized)

		return
	}

	content, err := h.service.Get(r.Context(), user, r.PathValue("id"))
	if errors.Is(err, service.ErrNotFound) {
		http.NotFound(w, r)

		return
	}

	if err != nil {
		log.Printf("get wiki: %v", err)
		message := "failed to get article"
		if errors.Is(err, service.ErrReadFile) {
			message = "failed to read article"
		}
		http.Error(w, message, http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	if _, err := w.Write(content); err != nil {
		log.Printf("write wiki article: %v", err)
	}
}
