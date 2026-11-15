package dialog

import (
	"backend/internal/buddy"
	"backend/internal/ollama"
	"backend/internal/users"
	"context"
	"database/sql"
	"errors"
	"testing"
)

type fakeRepository struct {
	chat     buddy.Chat
	messages []buddy.Message
	fail     string
}

var errFailure = errors.New("errFailure")

func (r *fakeRepository) AddChat(_ context.Context, chat buddy.Chat) (buddy.Chat, error) {
	r.chat = chat
	return chat, nil
}

func (r *fakeRepository) GetChat(context.Context, string) (buddy.Chat, error) {
	if r.fail == "get" {
		return buddy.Chat{}, errFailure
	}
	if r.fail == "missing" {
		return buddy.Chat{}, sql.ErrNoRows
	}
	return r.chat, nil
}

func (r *fakeRepository) AddMessage(_ context.Context, message buddy.Message) (buddy.Message, error) {
	if r.fail == message.Role {
		return buddy.Message{}, errFailure
	}
	message.ID = int64(len(r.messages) + 1)
	r.messages = append(r.messages, message)
	return message, nil
}

func (r *fakeRepository) ListMessages(context.Context, string) ([]buddy.Message, error) {
	if r.fail == "history" {
		return nil, errFailure
	}
	return r.messages, nil
}

type fakeAgent struct {
	history []ollama.Message
	err     error
	user    users.User
}

func (a *fakeAgent) Generate(_ context.Context, user users.User, history []ollama.Message, emit func(buddy.Event) error) (string, error) {
	a.history, a.user = history, user
	if a.err != nil {
		return "", a.err
	}
	if emit != nil {
		if err := emit(buddy.Event{Type: "text", Text: "Ответ"}); err != nil {
			return "", err
		}
	}
	return "Ответ", nil
}

func TestDialog(t *testing.T) {
	repo := &fakeRepository{}
	agent := &fakeAgent{}
	service := NewService(repo, agent)
	chat, err := service.CreateChat(t.Context(), "user", "Title")
	if err != nil || chat.ID == "" || chat.UserID != "user" || chat.Title != "Title" {
		t.Fatalf("chat: %+v, %v", chat, err)
	}
	user := users.User{ID: "user", Team: "backend"}
	for range 2 {
		var text string
		reply, err := service.Send(t.Context(), user, chat.ID, "Вопрос", func(event buddy.Event) error { text += event.Text; return nil })
		if err != nil || reply.Role != buddy.RoleAssistant || reply.Content != "Ответ" || reply.ChatID != chat.ID || text != "Ответ" {
			t.Fatalf("reply: %+v, %v", reply, err)
		}
	}
	if len(repo.messages) != 4 || len(agent.history) != 3 || agent.history[0].Role != "user" || agent.history[1].Role != "assistant" || agent.history[2].Content != "Вопрос" || agent.user != user {
		t.Fatalf("history: %+v", agent.history)
	}
}

func TestDialogFailures(t *testing.T) {
	for _, stage := range []string{"foreign", "missing", "get", "user", "history", "agent", "assistant", "empty", "emit"} {
		t.Run(stage, func(t *testing.T) {
			repo := &fakeRepository{chat: buddy.Chat{ID: "chat", UserID: "user"}, fail: stage}
			agent := &fakeAgent{}
			text := "question"
			if stage == "foreign" {
				repo.chat.UserID = "other"
			}
			if stage == "agent" {
				agent.err = errFailure
			}
			if stage == "empty" {
				text = " \n"
			}
			var emit func(buddy.Event) error
			if stage == "emit" {
				emit = func(buddy.Event) error { return errFailure }
			}
			_, err := NewService(repo, agent).Send(t.Context(), users.User{ID: "user", Team: "backend"}, "chat", text, emit)
			if err == nil {
				t.Fatal("expected error")
			}
			if stage == "foreign" || stage == "missing" {
				if !errors.Is(err, ErrNotFound) {
					t.Fatal(err)
				}
			}
			wantCount := 0
			if stage == "history" || stage == "agent" || stage == "assistant" || stage == "emit" {
				wantCount = 1
			}
			if len(repo.messages) != wantCount {
				t.Fatalf("stored messages: %v", repo.messages)
			}
		})
	}
}

type blockingAgent struct{ started, release chan struct{} }

func (a *blockingAgent) Generate(ctx context.Context, _ users.User, _ []ollama.Message, _ func(buddy.Event) error) (string, error) {
	close(a.started)
	select {
	case <-a.release:
		return "Ответ", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func TestConcurrentSend(t *testing.T) {
	repo := &fakeRepository{chat: buddy.Chat{ID: "chat", UserID: "user"}}
	agent := &blockingAgent{started: make(chan struct{}), release: make(chan struct{})}
	service := NewService(repo, agent)
	user := users.User{ID: "user"}
	done := make(chan error, 1)
	go func() { _, err := service.Send(t.Context(), user, "chat", "first", nil); done <- err }()
	<-agent.started
	_, err := service.Send(t.Context(), user, "chat", "second", nil)
	close(agent.release)
	firstErr := <-done
	if !errors.Is(err, ErrBusy) || firstErr != nil || len(repo.messages) != 2 {
		t.Fatalf("concurrent send: %v, %v, %+v", err, firstErr, repo.messages)
	}
	// The slot is released after completion.
	service.agent = &fakeAgent{}
	if _, err := service.Send(t.Context(), user, "chat", "next", nil); err != nil {
		t.Fatal(err)
	}
}
