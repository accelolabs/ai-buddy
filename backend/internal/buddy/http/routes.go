package buddyhttp

import (
	"net/http"

	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/platform/httpx"
)

func (h *Handler) Routes(login *auth.Auth) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/chats", h.create)
	mux.HandleFunc("GET /api/chats", h.list)
	mux.HandleFunc("GET /api/chats/{id}", h.history)
	mux.HandleFunc("POST /api/chats/{id}/messages", h.send)
	mux.HandleFunc("GET /api/runs/{id}/events", h.events)
	mux.HandleFunc("POST /api/runs/{id}/retry", h.retry)
	mux.HandleFunc("POST /api/actions/{id}/confirm", h.confirm)
	mux.HandleFunc("POST /api/actions/{id}/cancel", h.cancel)
	httpx.Fallbacks(mux, map[string][]string{"/api/chats": {"GET", "POST"}, "/api/chats/{id}": {"GET"}, "/api/chats/{id}/messages": {"POST"}, "/api/runs/{id}/events": {"GET"}, "/api/runs/{id}/retry": {"POST"}, "/api/actions/{id}/confirm": {"POST"}, "/api/actions/{id}/cancel": {"POST"}})
	return login.WithSession(login.RequireRoles(mux, auth.RoleIntern))
}
