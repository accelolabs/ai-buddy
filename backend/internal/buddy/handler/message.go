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
	ConfirmTicket(context.Context, users.User, string, int64, string, string) (buddy.Ticket, bool, error)
	ListChats(context.Context, users.User) ([]buddy.Chat, error)
	GetChat(context.Context, users.User, string) (buddy.Chat, []buddy.Message, error)
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
		ID        int64           `json:"id"`
		ChatID    string          `json:"chat_id"`
		Role      string          `json:"role"`
		Content   string          `json:"content"`
		CreatedAt time.Time       `json:"created_at"`
		Source    *sourceResponse `json:"source"`
		Ticket    *ticketResponse `json:"ticket"`
	}
)

type sourceResponse struct {
	DocumentID string `json:"document_id"`
	Ordinal    int    `json:"ordinal"`
	Title      string `json:"title"`
	Anchor     string `json:"anchor"`
	Quote      string `json:"quote"`
}

func sourceView(source *buddy.Source) *sourceResponse {
	if source == nil {
		return nil
	}
	return &sourceResponse{DocumentID: source.DocumentID, Ordinal: source.Ordinal, Title: source.Title, Anchor: source.Anchor, Quote: source.Quote}
}

func NewHandler(dialog dialogService) *Handler { return &Handler{dialog: dialog} }

// Send accepts a message, creates a chat when the route has no id, and streams the reply via SSE.
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
	stream := sseWriter{w: w}
	reply, err := h.dialog.Send(r.Context(), user, r.PathValue("id"), input.Content, func(event buddy.Event) error {
		switch event.Type {
		case "chat":
			if event.Chat == nil {
				return fmt.Errorf("missing chat in event")
			}
			return stream.send("chat", chatResponse(*event.Chat))
		case "state":
			return stream.send("state", stateResponse{State: event.State, Progress: event.Progress})
		case "source":
			return stream.send("source", sourceView(event.Source))
		case "ticket":
			return stream.send("ticket", ticketView(event.Ticket))
		case "text":
			return stream.send("text", textResponse{Text: event.Text})
		default:
			return fmt.Errorf("unknown agent event %q", event.Type)
		}
	})
	if err != nil {
		log.Printf("send dialog message: %v", err)
		if r.Context().Err() != nil {
			return
		}
		if stream.started {
			if err := stream.send("error", errorResponse{Error: "failed to generate reply"}); err != nil {
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
	if err := stream.send("done", messageView(reply)); err != nil {
		log.Printf("write dialog reply: %v", err)
	}
}
