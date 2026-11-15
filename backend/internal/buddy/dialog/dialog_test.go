package dialog

import (
	"backend/internal/buddy"
	"backend/internal/users"
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

type fakeRepository struct {
	chat         buddy.Chat
	messages     []buddy.Message
	fail         string
	listedUser   string
	historyCalls int
}

var errFailure = errors.New("errFailure")

func (r *fakeRepository) AddChat(_ context.Context, chat buddy.Chat) (buddy.Chat, error) {
	if r.fail == "chat" {
		return buddy.Chat{}, errFailure
	}
	r.chat = chat
	return chat, nil
}

func (r *fakeRepository) ListChats(_ context.Context, userID string) ([]buddy.Chat, error) {
	r.listedUser = userID
	if r.fail == "list" {
		return nil, errFailure
	}
	return []buddy.Chat{r.chat}, nil
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
	r.historyCalls++
	if r.fail == "history" {
		return nil, errFailure
	}
	return r.messages, nil
}

type fakeAgent struct {
	source  *buddy.Source
	history []buddy.Message
	err     error
	user    users.User
}

func (a *fakeAgent) Generate(_ context.Context, user users.User, history []buddy.Message, emit func(buddy.Event) error) (buddy.Answer, error) {
	a.history, a.user = history, user
	if a.err != nil {
		return buddy.Answer{}, a.err
	}
	if emit != nil {
		if err := emit(buddy.Event{Type: "text", Text: "Ответ"}); err != nil {
			return buddy.Answer{}, err
		}
	}
	return buddy.Answer{Content: "Ответ", Source: a.source}, nil
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

func (a *blockingAgent) Generate(ctx context.Context, _ users.User, _ []buddy.Message, _ func(buddy.Event) error) (buddy.Answer, error) {
	close(a.started)
	select {
	case <-a.release:
		return buddy.Answer{Content: "Ответ"}, nil
	case <-ctx.Done():
		return buddy.Answer{}, ctx.Err()
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

func TestSourceAfterSave(t *testing.T) {
	for _, stage := range []string{"success", "none", "agent", "assistant", "source-write"} {
		t.Run(stage, func(t *testing.T) {
			source := &buddy.Source{DocumentID: "doc", Ordinal: 2, Title: "Title", Anchor: "экватор", Quote: "Full quote"}
			repo := &fakeRepository{chat: buddy.Chat{ID: "chat", UserID: "user"}, fail: stage}
			agent := &fakeAgent{source: source}
			if stage == "none" {
				agent.source = nil
			}
			if stage == "agent" {
				agent.err = errFailure
			}
			var events []string
			reply, err := NewService(repo, agent).Send(t.Context(), users.User{ID: "user"}, "chat", "question", func(e buddy.Event) error {
				events = append(events, e.Type)
				if e.Type == "source" {
					if len(repo.messages) != 2 || repo.messages[1].Source != e.Source || *e.Source != *source {
						t.Fatal("source emitted before storage or differs")
					}
					if stage == "source-write" {
						return errFailure
					}
				}
				return nil
			})
			switch stage {
			case "success":
				if err != nil || reply.Source != source || len(events) != 2 || events[0] != "text" || events[1] != "source" {
					t.Fatalf("%+v %v %v", reply, err, events)
				}
			case "none":
				if err != nil || reply.Source != nil || len(events) != 1 {
					t.Fatalf("%+v %v %v", reply, err, events)
				}
			case "source-write":
				if !errors.Is(err, errFailure) || len(repo.messages) != 2 || repo.messages[1].Source != source {
					t.Fatal("saved answer lost on stream failure")
				}
			default:
				if err == nil || len(repo.messages) != 1 {
					t.Fatal(err, repo.messages)
				}
				for _, event := range events {
					if event == "source" {
						t.Fatal("source emitted on failure")
					}
				}
			}
		})
	}
}

func TestReadChats(t *testing.T) {
	for _, stage := range []string{"success", "foreign", "missing", "get", "history"} {
		t.Run(stage, func(t *testing.T) {
			repo := &fakeRepository{
				chat: buddy.Chat{ID: "chat", UserID: "user"}, fail: stage,
				messages: []buddy.Message{{ID: 1, Content: "question"}, {ID: 2, Content: "answer", Source: &buddy.Source{Title: "Title"}}},
			}
			if stage == "foreign" {
				repo.chat.UserID = "other"
			}
			service := NewService(repo, nil)
			chat, messages, err := service.GetChat(t.Context(), users.User{ID: "user"}, "chat")
			if stage == "success" {
				if err != nil || chat != repo.chat || len(messages) != 2 || messages[1].Source != repo.messages[1].Source {
					t.Fatalf("chat: %+v messages: %+v error: %v", chat, messages, err)
				}
			} else if stage == "foreign" || stage == "missing" {
				if !errors.Is(err, ErrNotFound) || repo.historyCalls != 0 {
					t.Fatalf("access: %v, history calls %d", err, repo.historyCalls)
				}
			} else if !errors.Is(err, errFailure) {
				t.Fatal(err)
			}
		})
	}
	for _, failure := range []string{"", "list"} {
		repo := &fakeRepository{chat: buddy.Chat{ID: "chat", UserID: "user"}, fail: failure}
		chats, err := NewService(repo, nil).ListChats(t.Context(), users.User{ID: "user"})
		if repo.listedUser != "user" || (failure == "" && (err != nil || len(chats) != 1)) || (failure == "list" && !errors.Is(err, errFailure)) {
			t.Fatalf("list: %+v, %v, user %s", chats, err, repo.listedUser)
		}
	}
}

func TestFirstMessageCreatesChat(t *testing.T) {
	for _, tc := range []struct{ text, title string }{
		{"Как получить доступ?", "Как получить доступ?"},
		{" \n  Первый\tвопрос  \n стажёра ", "Первый вопрос стажёра"},
		{strings.Repeat("Я🙂", 45), strings.Repeat("Я🙂", 40)},
	} {
		repo := &fakeRepository{}
		agent := &fakeAgent{}
		service := NewService(repo, agent)
		var events []string
		reply, err := service.Send(t.Context(), users.User{ID: "user"}, "", tc.text, func(event buddy.Event) error {
			events = append(events, event.Type)
			if event.Type == "chat" {
				if len(events) != 1 || event.Chat == nil || *event.Chat != repo.chat || event.Chat.Title != tc.title || len(repo.messages) != 1 || repo.messages[0].Content != tc.text {
					t.Fatalf("chat event before storage or wrong data: %+v, %+v", event, repo.messages)
				}
			}
			return nil
		})
		if err != nil || repo.chat.ID == "" || repo.chat.UserID != "user" || reply.ChatID != repo.chat.ID || len(repo.messages) != 2 || len(agent.history) != 1 || strings.Join(events, ",") != "chat,text" {
			t.Fatalf("new chat: %+v, %+v, events: %v, error: %v", repo.chat, reply, events, err)
		}
		// Follow-up messages reuse the same chat and do not change its title or emit chat.
		if _, err := service.Send(t.Context(), users.User{ID: "user"}, repo.chat.ID, "Следующий вопрос", func(event buddy.Event) error {
			if event.Type == "chat" {
				t.Fatal("chat emitted for existing chat")
			}
			return nil
		}); err != nil || repo.chat.Title != tc.title || len(repo.messages) != 4 {
			t.Fatalf("follow-up: %v %+v", err, repo.chat)
		}
	}
}

func TestFirstMessageFailures(t *testing.T) {
	for _, stage := range []string{"empty", "chat", "user", "history", "agent", "assistant", "emit"} {
		t.Run(stage, func(t *testing.T) {
			repo := &fakeRepository{fail: stage}
			agent := &fakeAgent{}
			if stage == "agent" {
				agent.err = errFailure
			}
			text := "Первый вопрос"
			if stage == "empty" {
				text = " \n\t"
			}
			var events []string
			service := NewService(repo, agent)
			_, err := service.Send(t.Context(), users.User{ID: "user"}, "", text, func(event buddy.Event) error {
				events = append(events, event.Type)
				if stage == "emit" {
					return errFailure
				}
				return nil
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if stage == "empty" || stage == "chat" {
				if repo.chat.ID != "" || len(repo.messages) != 0 || len(events) != 0 {
					t.Fatal("chat created on invalid input or failed insert")
				}
			} else if stage == "user" {
				if len(events) != 0 {
					t.Fatal("chat event emitted before first message was saved")
				}
			} else if len(repo.messages) != 1 || repo.messages[0].Content != text || len(events) == 0 || events[0] != "chat" {
				t.Fatalf("lost question or chat event: %+v %v", repo.messages, events)
			}
			if _, busy := service.active.Load(repo.chat.ID); busy {
				t.Fatal("chat remains locked after failure")
			}
		})
	}
}

func TestFirstMessageLocksChat(t *testing.T) {
	repo := &fakeRepository{}
	agent := &blockingAgent{started: make(chan struct{}), release: make(chan struct{})}
	service := NewService(repo, agent)
	user := users.User{ID: "user"}
	done := make(chan error, 1)
	go func() { _, err := service.Send(t.Context(), user, "", "Первый вопрос", nil); done <- err }()
	<-agent.started
	_, err := service.Send(t.Context(), user, repo.chat.ID, "Второй вопрос", nil)
	close(agent.release)
	firstErr := <-done
	if !errors.Is(err, ErrBusy) || firstErr != nil || len(repo.messages) != 2 {
		t.Fatalf("concurrent first message: %v, %v, %+v", err, firstErr, repo.messages)
	}
}
