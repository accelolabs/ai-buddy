package ticketshttp

import (
	"ai-buddy/internal/users"
	"net/http"
)

func Register(mux *http.ServeMux, login *users.Selector, h *Handler) {
	mux.Handle("GET /api/tickets", login.RequireRoles(http.HandlerFunc(h.Sent), users.RoleIntern))
	mux.Handle("POST /api/tickets", login.RequireRoles(http.HandlerFunc(h.Create), users.RoleIntern))
	mux.Handle("GET /api/tickets/incoming", login.RequireRoles(http.HandlerFunc(h.Incoming), users.RoleMentor))
}
