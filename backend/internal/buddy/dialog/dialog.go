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
	GetChat(context.Context, string) (buddy.Chat, error)
	AddMessage(context.Context, buddy.Message) (buddy.Message, error)
	ListMessages(context.Context, string) ([]buddy.Message, error)
}

type agent interface {
	Generate(context.Context, users.User, []buddy.Message, func(buddy.Event) error) (buddy.Answer, error)
}

type Service struct {
	active     sync.Map
	repository repository
	agent      agent
}

func NewService(repository repository, agent agent) *Service {
	return &Service{repository: repository, agent: agent}
}

func (s *Service) CreateChat(ctx context.Context, userID, title string) (buddy.Chat, error) {
	return s.repository.AddChat(ctx, buddy.Chat{ID: rand.Text(), UserID: userID, Title: title})
}

// Send requires a trusted current user.
// Concurrent sends to one chat are rejected within this service instance.
func (s *Service) Send(ctx context.Context, user users.User, chatID, text string, emit func(buddy.Event) error) (buddy.Message, error) {
	if strings.TrimSpace(text) == "" {
		return buddy.Message{}, fmt.Errorf("message must not be empty")
	}
	chat, err := s.repository.GetChat(ctx, chatID)
	if errors.Is(err, sql.ErrNoRows) {
		return buddy.Message{}, ErrNotFound
	}
	if err != nil {
		return buddy.Message{}, fmt.Errorf("get dialog chat: %w", err)
	}
	if chat.UserID != user.ID {
		return buddy.Message{}, ErrNotFound
	}
	if _, busy := s.active.LoadOrStore(chatID, struct{}{}); busy {
		return buddy.Message{}, ErrBusy
	}
	defer s.active.Delete(chatID)
	_, err = s.repository.AddMessage(ctx, buddy.Message{ChatID: chatID, Role: buddy.RoleUser, Content: text})
	if err != nil {
		return buddy.Message{}, fmt.Errorf("save user message: %w", err)
	}
	history, err := s.repository.ListMessages(ctx, chatID)
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
