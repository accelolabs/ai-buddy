package handler

import (
	"backend/internal/buddy"
	"backend/internal/buddy/dialog"
	"backend/internal/users"
	"backend/internal/users/middleware"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type dialogService interface {
	Send(context.Context, users.User, string, string, func(buddy.Event) error) (buddy.Message, error)
}

type Handler struct{ dialog dialogService }

type (
	messageRequest struct {
		Content string `json:"content"`
	}
	stateResponse struct {
		State    string `json:"state"`
		Progress string `json:"progress,omitempty"`
	}
)

type (
	textResponse struct {
		Text string `json:"text"`
	}
	errorResponse struct {
		Error string `json:"error"`
	}
	messageResponse struct {
		ID        int64     `json:"id"`
		ChatID    string    `json:"chat_id"`
		Role      string    `json:"role"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
	}
)

func NewHandler(dialog dialogService) *Handler { return &Handler{dialog: dialog} }

// Send accepts a message and streams agent activity and its persisted reply via SSE.
func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "select a user", http.StatusUnauthorized)
		return
	}
	var input messageRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid message", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF || strings.TrimSpace(input.Content) == "" {
		http.Error(w, "invalid message", http.StatusBadRequest)
		return
	}
	if _, ok := w.(http.Flusher); !ok {
		http.Error(w, "streaming unavailable", http.StatusInternalServerError)
		return
	}
	started := false
	send := func(event string, data any) error {
		body, err := json.Marshal(data)
		if err != nil {
			return err
		}
		if !started {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("X-Accel-Buffering", "no")
			started = true
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, body); err != nil {
			return err
		}
		return http.NewResponseController(w).Flush()
	}
	reply, err := h.dialog.Send(r.Context(), user, r.PathValue("id"), input.Content, func(event buddy.Event) error {
		switch event.Type {
		case "state":
			return send("state", stateResponse{State: event.State, Progress: event.Progress})
		case "text":
			return send("text", textResponse{Text: event.Text})
		default:
			return fmt.Errorf("unknown agent event %q", event.Type)
		}
	})
	if err != nil {
		log.Printf("send dialog message: %v", err)
		if r.Context().Err() != nil {
			return
		}
		if started {
			if err := send("error", errorResponse{Error: "failed to generate reply"}); err != nil {
				log.Printf("write dialog error: %v", err)
			}
		} else if errors.Is(err, dialog.ErrNotFound) {
			http.NotFound(w, r)
		} else if errors.Is(err, dialog.ErrBusy) {
			http.Error(w, "chat is busy", http.StatusConflict)
		} else {
			http.Error(w, "failed to send message", http.StatusInternalServerError)
		}
		return
	}
	if err := send("done", messageResponse{ID: reply.ID, ChatID: reply.ChatID, Role: reply.Role, Content: reply.Content, CreatedAt: reply.CreatedAt}); err != nil {
		log.Printf("write dialog reply: %v", err)
	}
}
