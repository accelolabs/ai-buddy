package buddyhttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/platform/httpx"
)

type Handler struct {
	Service   *buddy.Service
	Poll      time.Duration
	Heartbeat time.Duration
}

func New(s *buddy.Service) *Handler { return &Handler{s, 250 * time.Millisecond, 15 * time.Second} }
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
func fail(w http.ResponseWriter, e error) {
	code := 500
	msg := "service unavailable"
	switch {
	case errors.Is(e, buddy.ErrNotFound):
		code = 404
		msg = e.Error()
	case errors.Is(e, buddy.ErrConflict):
		code = 409
		msg = e.Error()
	case errors.Is(e, buddy.ErrInvalid):
		code = 400
		msg = e.Error()
	}
	httpx.JSON(w, code, map[string]string{"error": msg})
}
func owner(r *http.Request) auth.User { u, _ := auth.CurrentUser(r.Context()); return u }

type chatDTO struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

func chat(c buddy.Chat) chatDTO { return chatDTO{c.ID, c.Title, c.CreatedAt} }

type runDTO struct {
	ID        string `json:"id"`
	ChatID    string `json:"chat_id"`
	Status    string `json:"status"`
	Error     string `json:"error"`
	EventsURL string `json:"events_url"`
}

func run(r buddy.Run) runDTO {
	return runDTO{r.ID, r.ChatID, r.Status, r.Error, "/api/runs/" + r.ID + "/events"}
}

type actionDTO struct {
	ID     string          `json:"id"`
	RunID  string          `json:"run_id"`
	Kind   string          `json:"kind"`
	Status string          `json:"status"`
	Fields json.RawMessage `json:"fields"`
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

func action(a buddy.Action) actionDTO {
	return actionDTO{a.ID, a.RunID, a.Kind, a.Status, a.Fields, a.Result, a.Error}
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	c, e := h.Service.Repo.CreateChat(r.Context(), owner(r).ID)
	if e != nil {
		fail(w, e)
		return
	}
	httpx.JSON(w, 201, chat(c))
}
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	all, e := h.Service.Repo.Chats(r.Context(), owner(r).ID)
	if e != nil {
		fail(w, e)
		return
	}
	out := []chatDTO{}
	for _, c := range all {
		out = append(out, chat(c))
	}
	httpx.JSON(w, 200, map[string]any{"chats": out})
}
func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.Repo.History(r.Context(), owner(r).ID, r.PathValue("id"))
	if e != nil {
		fail(w, e)
		return
	}
	type message struct {
		ID        string    `json:"id"`
		RunID     string    `json:"run_id"`
		Role      string    `json:"role"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
	}
	messages := []message{}
	for _, m := range v.Messages {
		messages = append(messages, message{m.ID, m.RunID, m.Role, m.Content, m.CreatedAt})
	}
	runs := []runDTO{}
	for _, v := range v.Runs {
		runs = append(runs, run(v))
	}
	actions := []actionDTO{}
	for _, v := range v.Actions {
		actions = append(actions, action(v))
	}
	type card struct {
		RunID string          `json:"run_id"`
		Kind  string          `json:"kind"`
		Data  json.RawMessage `json:"data"`
	}
	cards := []card{}
	for _, c := range v.Cards {
		cards = append(cards, card{c.RunID, c.Kind, c.Data})
	}
	var latest *runDTO
	if len(runs) > 0 {
		latest = &runs[len(runs)-1]
	}
	httpx.JSON(w, 200, map[string]any{"chat": chat(v.Chat), "messages": messages, "runs": runs, "latest_run": latest, "actions": actions, "cards": cards})
}
func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Content string `json:"content"`
	}
	if httpx.Decode(w, r, &in) != nil {
		fail(w, buddy.ErrInvalid)
		return
	}
	v, e := h.Service.Send(r.Context(), owner(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), in.Content)
	if e != nil {
		fail(w, e)
		return
	}
	httpx.JSON(w, 202, run(v))
}
func (h *Handler) retry(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.Retry(r.Context(), owner(r), r.PathValue("id"))
	if e != nil {
		fail(w, e)
		return
	}
	httpx.JSON(w, 202, run(v))
}
func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Fields json.RawMessage `json:"fields"`
	}
	if httpx.Decode(w, r, &in) != nil {
		fail(w, buddy.ErrInvalid)
		return
	}
	h.resolve(w, r, in.Fields, false)
}
func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) { h.resolve(w, r, nil, true) }
func (h *Handler) resolve(w http.ResponseWriter, r *http.Request, fields json.RawMessage, cancel bool) {
	a, e := h.Service.Confirm(r.Context(), owner(r), r.PathValue("id"), fields, cancel)
	if e != nil {
		fail(w, e)
		return
	}
	httpx.JSON(w, 200, action(a))
}
func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	after := int64(0)
	if raw := r.Header.Get("Last-Event-ID"); raw != "" {
		var e error
		after, e = strconv.ParseInt(raw, 10, 64)
		if e != nil || after < 0 {
			fail(w, buddy.ErrInvalid)
			return
		}
	}
	events, current, e := h.Service.Repo.Events(r.Context(), owner(r).ID, r.PathValue("id"), after)
	if e != nil {
		fail(w, e)
		return
	}
	if _, ok := w.(http.Flusher); !ok {
		fail(w, errors.New("streaming unavailable"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	controller := http.NewResponseController(w)
	write := func(text string) bool {
		_ = controller.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if _, e := fmt.Fprint(w, text); e != nil {
			return false
		}
		return controller.Flush() == nil
	}
	poll := time.NewTicker(h.Poll)
	defer poll.Stop()
	heartbeat := time.NewTicker(h.Heartbeat)
	defer heartbeat.Stop()
	for {
		for _, event := range events {
			if !write(fmt.Sprintf("id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Kind, event.Data)) {
				return
			}
			after = event.ID
		}
		if !current.Active() {
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if !write(": heartbeat\n\n") {
				return
			}
		case <-poll.C:
		}
		events, current, e = h.Service.Repo.Events(r.Context(), owner(r).ID, current.ID, after)
		if e != nil {
			return
		}
	}
}
