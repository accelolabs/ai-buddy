package wikihttp

import (
	"net/http"

	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/platform/httpx"
)

func Routes(login *auth.Auth, h *Handler, key string) http.Handler {
	mux := http.NewServeMux()
	browser := http.NewServeMux()
	browser.HandleFunc("POST /api/search", h.Search)
	browser.HandleFunc("GET /api/documents", h.List)
	browser.HandleFunc("GET /api/documents/{id}", h.Get)
	httpx.Fallbacks(browser, map[string][]string{
		"/api/search":         {"POST"},
		"/api/documents":      {"GET"},
		"/api/documents/{id}": {"GET"},
	})
	mux.Handle("/api/", login.WithSession(login.RequireUser(browser)))
	mux.Handle("POST /internal/users/{id}/search", httpx.KeyAuth(http.HandlerFunc(h.Search), key, "", false))
	httpx.Fallbacks(mux, map[string][]string{"/internal/users/{id}/search": {"POST"}})
	return mux
}
