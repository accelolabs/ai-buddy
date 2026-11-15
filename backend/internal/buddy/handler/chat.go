package handler

import (
	"backend/internal/buddy"
	"backend/internal/buddy/dialog"
	"backend/internal/users/middleware"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

type chatResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type chatHistoryResponse struct {
	chatResponse
	Messages []messageResponse `json:"messages"`
}

func (h *Handler) ListChats(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "select a user", http.StatusUnauthorized)
		return
	}
	chats, err := h.dialog.ListChats(r.Context(), user)
	if err != nil {
		log.Printf("list dialog chats: %v", err)
		http.Error(w, "failed to list chats", http.StatusInternalServerError)
		return
	}
	response := make([]chatResponse, len(chats))
	for i, chat := range chats {
		response[i] = chatResponse(chat)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write chats response: %v", err)
	}
}

func (h *Handler) GetChat(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "select a user", http.StatusUnauthorized)
		return
	}
	chat, messages, err := h.dialog.GetChat(r.Context(), user, r.PathValue("id"))
	if errors.Is(err, dialog.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("get dialog chat: %v", err)
		http.Error(w, "failed to get chat", http.StatusInternalServerError)
		return
	}
	response := chatHistoryResponse{chatResponse: chatResponse(chat), Messages: make([]messageResponse, len(messages))}
	for i, message := range messages {
		response.Messages[i] = messageView(message)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write chat response: %v", err)
	}
}

func messageView(message buddy.Message) messageResponse {
	return messageResponse{ID: message.ID, ChatID: message.ChatID, Role: message.Role, Content: message.Content, CreatedAt: message.CreatedAt, Source: sourceView(message.Source)}
}
