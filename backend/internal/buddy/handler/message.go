package handler

import (
	"backend/internal/buddy"
	"backend/internal/users"
	"backend/internal/users/middleware"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type generator interface {
	Generate(context.Context, users.User, []buddy.Message, func(buddy.Event) error) (buddy.Answer, error)
}

type Handler struct {
	agent        generator
	historyLimit int
}

type inputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messageRequest struct {
	Messages []inputMessage `json:"messages"`
}

type stateResponse struct {
	State    string `json:"state"`
	Progress string `json:"progress,omitempty"`
}

type textResponse struct {
	Text string `json:"text"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type messageResponse struct {
	Content string          `json:"content"`
	Source  *sourceResponse `json:"source"`
	Ticket  *ticketResponse `json:"ticket"`
}

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

func NewHandler(agent generator, historyLimit int) *Handler {
	return &Handler{agent: agent, historyLimit: historyLimit}
}

// Send generates a reply using only the context supplied in this request.
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
	if err := decoder.Decode(new(any)); err != io.EOF || len(input.Messages) == 0 {
		http.Error(w, "invalid message", http.StatusBadRequest)
		return
	}
	for _, message := range input.Messages {
		if (message.Role != buddy.RoleUser && message.Role != buddy.RoleAssistant) || strings.TrimSpace(message.Content) == "" {
			http.Error(w, "invalid message", http.StatusBadRequest)
			return
		}
	}
	if input.Messages[len(input.Messages)-1].Role != buddy.RoleUser {
		http.Error(w, "invalid message", http.StatusBadRequest)
		return
	}
	history := make([]buddy.Message, len(input.Messages))
	for i, message := range input.Messages {
		history[i] = buddy.Message{Role: message.Role, Content: message.Content}
	}
	if len(history) > h.historyLimit {
		history = history[len(history)-h.historyLimit:]
	}
	if _, ok := w.(http.Flusher); !ok {
		http.Error(w, "streaming unavailable", http.StatusInternalServerError)
		return
	}
	stream := sseWriter{w: w}
	reply, err := h.agent.Generate(r.Context(), user, history, func(event buddy.Event) error {
		switch event.Type {
		case "state":
			return stream.send("state", stateResponse{State: event.State, Progress: event.Progress})
		case "text":
			return stream.send("text", textResponse{Text: event.Text})
		default:
			return fmt.Errorf("unknown agent event %q", event.Type)
		}
	})
	if err != nil {
		log.Printf("generate buddy reply: %v", err)
		if r.Context().Err() != nil {
			return
		}
		if stream.started {
			if err := stream.send("error", errorResponse{Error: "failed to generate reply"}); err != nil {
				log.Printf("write buddy error: %v", err)
			}
		} else {
			http.Error(w, "failed to send message", http.StatusInternalServerError)
		}
		return
	}
	if reply.Source != nil {
		if err := stream.send("source", sourceView(reply.Source)); err != nil {
			log.Printf("write source: %v", err)
			return
		}
	}
	if reply.Ticket != nil {
		if err := stream.send("ticket", ticketView(reply.Ticket)); err != nil {
			log.Printf("write ticket: %v", err)
			return
		}
	}
	if err := stream.send("done", messageResponse{Content: reply.Content, Source: sourceView(reply.Source), Ticket: ticketView(reply.Ticket)}); err != nil {
		log.Printf("write buddy reply: %v", err)
	}
}
