package buddyhttp

import (
	"net/http"

	"ai-buddy/internal/users"
)

func (h *Handler) Register(mux *http.ServeMux, login *users.Selector) {
	mux.Handle("POST /api/chats", login.RequireRoles(http.HandlerFunc(h.create), users.RoleIntern))
	mux.Handle("GET /api/chats", login.RequireRoles(http.HandlerFunc(h.list), users.RoleIntern))
	mux.Handle("GET /api/chats/{id}", login.RequireRoles(http.HandlerFunc(h.history), users.RoleIntern))
	mux.Handle("POST /api/chats/{id}/messages", login.RequireRoles(http.HandlerFunc(h.send), users.RoleIntern))
	mux.Handle("GET /api/runs/{id}/events", login.RequireRoles(http.HandlerFunc(h.events), users.RoleIntern))
	mux.Handle("POST /api/runs/{id}/retry", login.RequireRoles(http.HandlerFunc(h.retry), users.RoleIntern))
	mux.Handle("POST /api/actions/{id}/confirm", login.RequireRoles(http.HandlerFunc(h.confirm), users.RoleIntern))
	mux.Handle("POST /api/actions/{id}/cancel", login.RequireRoles(http.HandlerFunc(h.cancel), users.RoleIntern))
}
