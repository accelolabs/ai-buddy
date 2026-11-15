package handler

import (
	"backend/internal/users/middleware"
	"backend/internal/wiki"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
)

type documentReader interface {
	List(context.Context) ([]wiki.Document, error)
	Get(context.Context, string) (wiki.Document, error)
}

type Handler struct {
	repository documentReader
	root       string
}

func NewHandler(repository documentReader, root string) *Handler {
	return &Handler{repository: repository, root: root}
}

type documentResponse struct {
	ID    string `json:"id"`
	Team  string `json:"team"`
	Title string `json:"title"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "select a user", http.StatusUnauthorized)
		return
	}
	documents, err := h.repository.List(r.Context())
	if err != nil {
		log.Printf("list wiki: %v", err)
		http.Error(w, "failed to list articles", http.StatusInternalServerError)
		return
	}
	response := make([]documentResponse, 0, len(documents))
	for _, document := range documents {
		if document.Team == "" || document.Team == user.Team {
			response = append(response, documentResponse{
				ID:    document.ID,
				Team:  document.Team,
				Title: document.Title,
			})
		}
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
	document, err := h.repository.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("get wiki: %v", err)
		http.Error(w, "failed to get article", http.StatusInternalServerError)
		return
	}

	if document.Team != "" && document.Team != user.Team {
		http.NotFound(w, r)
		return
	}

	root, err := os.OpenRoot(h.root)
	if err != nil {
		log.Printf("open knowledge base: %v", err)
		http.Error(w, "failed to read article", http.StatusInternalServerError)
		return
	}
	defer func() { _ = root.Close() }()
	content, err := root.ReadFile(document.Path)
	if errors.Is(err, fs.ErrNotExist) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("read wiki %q: %v", document.ID, err)
		http.Error(w, "failed to read article", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	if _, err := w.Write(content); err != nil {
		log.Printf("write wiki article: %v", err)
	}
}
