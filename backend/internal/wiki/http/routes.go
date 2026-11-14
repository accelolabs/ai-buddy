package wikihttp

import (
	"ai-buddy/internal/users"
	"net/http"
)

func Register(mux *http.ServeMux, login *users.Selector, h *Handler) {
	mux.Handle("POST /api/search", login.RequireUser(http.HandlerFunc(h.Search)))
	mux.Handle("GET /api/documents", login.RequireUser(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/documents/{id}", login.RequireUser(http.HandlerFunc(h.Get)))
}
