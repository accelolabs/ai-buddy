package buddy

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/users"
)

type Handler struct {
	Service   *Service
	Poll      time.Duration
	Heartbeat time.Duration
}

func NewHandler(s *Service) *Handler { return &Handler{s, 250 * time.Millisecond, 15 * time.Second} }

func fail(w http.ResponseWriter, e error) {
	code := 500
	msg := "service unavailable"
	switch {
	case errors.Is(e, ErrNotFound):
		code = 404
		msg = e.Error()
	case errors.Is(e, ErrConflict):
		code = 409
		msg = e.Error()
	case errors.Is(e, ErrInvalid):
		code = 400
		msg = e.Error()
	}
	httpx.JSON(w, code, map[string]string{"error": msg})
}

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

type runDTO struct {
	ID        string `json:"id"`
	ChatID    string `json:"chat_id"`
	Status    string `json:"status"`
	Error     string `json:"error"`
	EventsURL string `json:"events_url"`
}

func run(r Run) runDTO {
	return runDTO{r.ID, r.ChatID, r.Status, r.Error, "/api/runs/" + r.ID + "/events"}
}

type sendRequest struct {
	Content string `json:"content"`
}
type confirmRequest struct {
	Fields json.RawMessage `json:"fields"`
}
