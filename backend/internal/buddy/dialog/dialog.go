package dialog

import (
	"backend/internal/buddy"
	"backend/internal/users"
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	ErrNotFound = errors.New("chat not found")
	ErrBusy     = errors.New("chat is already generating a response")
)

type repository interface {
	AddChat(context.Context, buddy.Chat) (buddy.Chat, error)
	ListChats(context.Context, string) ([]buddy.Chat, error)
	GetChat(context.Context, string) (buddy.Chat, error)
	AddMessage(context.Context, buddy.Message) (buddy.Message, error)
	ListMessages(context.Context, string) ([]buddy.Message, error)
	ListRecentMessages(context.Context, string, int) ([]buddy.Message, error)
}

type agent interface {
	Generate(context.Context, users.User, []buddy.Message, func(buddy.Event) error) (buddy.Answer, error)
}

type Service struct {
	active       sync.Map
	repository   repository
	agent        agent
	historyLimit int
}

func NewService(repository repository, agent agent, historyLimit int) *Service {
	return &Service{repository: repository, agent: agent, historyLimit: historyLimit}
}

func (s *Service) ownedChat(ctx context.Context, user users.User, id string) (buddy.Chat, error) {
	chat, err := s.repository.GetChat(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return buddy.Chat{}, ErrNotFound
	}
	if err != nil {
		return buddy.Chat{}, fmt.Errorf("get dialog chat: %w", err)
	}
	if chat.UserID != user.ID {
		return buddy.Chat{}, ErrNotFound
	}
	return chat, nil
}

// ListChats returns only chats owned by the trusted current user.
func (s *Service) ListChats(ctx context.Context, user users.User) ([]buddy.Chat, error) {
	return s.repository.ListChats(ctx, user.ID)
}

// GetChat returns the owned chat and its complete persisted history.
func (s *Service) GetChat(ctx context.Context, user users.User, id string) (buddy.Chat, []buddy.Message, error) {
	chat, err := s.ownedChat(ctx, user, id)
	if err != nil {
		return buddy.Chat{}, nil, err
	}
	messages, err := s.repository.ListMessages(ctx, id)
	if err != nil {
		return buddy.Chat{}, nil, fmt.Errorf("load dialog history: %w", err)
	}
	return chat, messages, nil
}

// Send requires a trusted current user. An empty chatID creates a chat from the first message.
// Concurrent sends to one chat are rejected within this service instance.
func (s *Service) Send(ctx context.Context, user users.User, chatID, text string, emit func(buddy.Event) error) (buddy.Message, error) {
	if strings.TrimSpace(text) == "" {
		return buddy.Message{}, fmt.Errorf("message must not be empty")
	}
	newChat := chatID == ""
	var chat buddy.Chat
	var err error
	if newChat {
		title := []rune(strings.Join(strings.Fields(text), " "))
		chat, err = s.repository.AddChat(ctx, buddy.Chat{ID: rand.Text(), UserID: user.ID, Title: string(title[:min(len(title), 80)])})
		if err != nil {
			return buddy.Message{}, fmt.Errorf("create dialog chat: %w", err)
		}
		chatID = chat.ID
	} else {
		chat, err = s.ownedChat(ctx, user, chatID)
		if err != nil {
			return buddy.Message{}, err
		}
	}
	if _, busy := s.active.LoadOrStore(chatID, struct{}{}); busy {
		return buddy.Message{}, ErrBusy
	}
	defer s.active.Delete(chatID)
	_, err = s.repository.AddMessage(ctx, buddy.Message{ChatID: chatID, Role: buddy.RoleUser, Content: text})
	if err != nil {
		return buddy.Message{}, fmt.Errorf("save user message: %w", err)
	}
	if newChat && emit != nil {
		if err := emit(buddy.Event{Type: "chat", Chat: &chat}); err != nil {
			return buddy.Message{}, fmt.Errorf("emit new chat: %w", err)
		}
	}
	history, err := s.repository.ListRecentMessages(ctx, chatID, s.historyLimit)
	if err != nil {
		return buddy.Message{}, fmt.Errorf("load dialog history: %w", err)
	}
	answer, err := s.agent.Generate(ctx, user, history, emit)
	if err != nil {
		return buddy.Message{}, fmt.Errorf("generate dialog answer: %w", err)
	}
	message, err := s.repository.AddMessage(ctx, buddy.Message{ChatID: chatID, Role: buddy.RoleAssistant, Content: answer.Content, Source: answer.Source})
	if err != nil {
		return buddy.Message{}, fmt.Errorf("save assistant message: %w", err)
	}
	if message.Source != nil && emit != nil {
		if err := emit(buddy.Event{Type: "source", Source: message.Source}); err != nil {
			return buddy.Message{}, fmt.Errorf("emit source: %w", err)
		}
	}
	return message, nil
}
