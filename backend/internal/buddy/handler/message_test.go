package handler

import (
	"backend/internal/buddy"
	"backend/internal/buddy/dialog"
	"backend/internal/users"
	"backend/internal/users/middleware"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type userStore struct{}

func (userStore) Get(context.Context, string) (users.User, error) {
	return users.User{ID: "user", Team: "backend"}, nil
}

type fakeDialog struct {
	confirm func(context.Context, users.User, string, int64, string, string) (buddy.Ticket, bool, error)
	calls   int
	list    func(context.Context, users.User) ([]buddy.Chat, error)
	get     func(context.Context, users.User, string) (buddy.Chat, []buddy.Message, error)
	send    func(context.Context, users.User, string, string, func(buddy.Event) error) (buddy.Message, error)
}

func (d *fakeDialog) ListChats(ctx context.Context, user users.User) ([]buddy.Chat, error) {
	d.calls++
	return d.list(ctx, user)
}

func (d *fakeDialog) GetChat(ctx context.Context, user users.User, id string) (buddy.Chat, []buddy.Message, error) {
	d.calls++
	return d.get(ctx, user, id)
}

func (d *fakeDialog) Send(ctx context.Context, user users.User, id, text string, emit func(buddy.Event) error) (buddy.Message, error) {
	d.calls++
	return d.send(ctx, user, id, text, emit)
}

func request(t *testing.T, service *fakeDialog, body string, auth bool) *httptest.ResponseRecorder {
	t.Helper()
	return messageRequestAt(service, "/api/chats/chat/messages", body, auth)
}

func messageRequestAt(service *fakeDialog, target, body string, auth bool) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	h := middleware.Middleware(userStore{})(http.HandlerFunc(NewHandler(service).Send))
	mux.Handle("POST /api/chats/{id}/messages", h)
	mux.Handle("POST /api/chats/messages", h)
	r := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	if auth {
		r.AddCookie(&http.Cookie{Name: "demo_user", Value: "user"})
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestSendStream(t *testing.T) {
	service := &fakeDialog{send: func(_ context.Context, user users.User, id, text string, emit func(buddy.Event) error) (buddy.Message, error) {
		if user.ID != "user" || user.Team != "backend" || id != "chat" || text != "Вопрос" {
			t.Fatal("wrong dialog inputs")
		}
		for _, event := range []buddy.Event{
			{Type: "state", State: "thinking"},
			{Type: "state", State: "progress", Progress: "search_documents"},
			{Type: "state", State: "answering"},
			{Type: "text", Text: "Ответ\nновая строка"},
		} {
			if err := emit(event); err != nil {
				return buddy.Message{}, err
			}
		}
		return buddy.Message{ID: 7, ChatID: id, Role: buddy.RoleAssistant, Content: "Ответ\nновая строка"}, nil
	}}
	w := request(t, service, `{"content":"Вопрос"}`, true)
	if w.Code != 200 || w.Header().Get("Content-Type") != "text/event-stream" || !w.Flushed {
		t.Fatalf("response: %d %v", w.Code, w.Header())
	}
	body := w.Body.String()
	last := -1
	for _, part := range []string{`"state":"thinking"`, `"state":"progress","progress":"search_documents"`, `"state":"answering"`, `event: text`, `"text":"Ответ\nновая строка"`, `event: done`, `"id":7`} {
		index := strings.Index(body, part)
		if index < last || index < 0 {
			t.Fatalf("missing/out of order %s: %s", part, body)
		}
		last = index
	}
}

func TestSendValidation(t *testing.T) {
	for _, body := range []string{`broken`, `{}`, `{"content":" "}`, `{"content":3}`, `{"content":"q"} {}`, `{"content":"q"} trailing`} {
		service := &fakeDialog{}
		w := request(t, service, body, true)
		if w.Code != 400 || service.calls != 0 {
			t.Fatalf("%s: %d, %d", body, w.Code, service.calls)
		}
	}
	service := &fakeDialog{}
	if w := request(t, service, `{"content":"q"}`, false); w.Code != 401 || service.calls != 0 {
		t.Fatal(w.Code)
	}
}

func TestSendErrors(t *testing.T) {
	for _, test := range []struct {
		err    error
		status int
	}{{dialog.ErrNotFound, 404}, {dialog.ErrBusy, 409}, {errors.New("database secret"), 500}} {
		service := &fakeDialog{send: func(context.Context, users.User, string, string, func(buddy.Event) error) (buddy.Message, error) {
			return buddy.Message{}, test.err
		}}
		w := request(t, service, `{"content":"q"}`, true)
		if w.Code != test.status || strings.Contains(w.Body.String(), "secret") {
			t.Fatalf("%d: %s", w.Code, w.Body.String())
		}
	}
	service := &fakeDialog{send: func(_ context.Context, _ users.User, _, _ string, emit func(buddy.Event) error) (buddy.Message, error) {
		if err := emit(buddy.Event{Type: "text", Text: "partial"}); err != nil {
			return buddy.Message{}, err
		}
		return buddy.Message{}, errors.New("internal secret")
	}}
	w := request(t, service, `{"content":"q"}`, true)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "event: error") || strings.Contains(w.Body.String(), "event: done") || strings.Contains(w.Body.String(), "event: source") || strings.Contains(w.Body.String(), "secret") {
		t.Fatal(w.Body.String())
	}
}

func TestSourceStream(t *testing.T) {
	source := &buddy.Source{DocumentID: "doc", Ordinal: 2, Title: "Памятка", Anchor: "экватор", Quote: "Full quote\nSecond line"}
	for _, attached := range []bool{false, true} {
		service := &fakeDialog{send: func(_ context.Context, _ users.User, id, _ string, emit func(buddy.Event) error) (buddy.Message, error) {
			if err := emit(buddy.Event{Type: "text", Text: "Answer"}); err != nil {
				return buddy.Message{}, err
			}
			var selected *buddy.Source
			if attached {
				selected = source
				if err := emit(buddy.Event{Type: "source", Source: selected}); err != nil {
					return buddy.Message{}, err
				}
			}
			return buddy.Message{ID: 42, ChatID: id, Role: buddy.RoleAssistant, Content: "Answer", Source: selected}, nil
		}}
		body := request(t, service, `{"content":"q"}`, true).Body.String()
		parts := strings.Split(strings.TrimSpace(body), "\n\n")
		if attached {
			if len(parts) != 3 || !strings.HasPrefix(parts[0], "event: text\n") || !strings.HasPrefix(parts[1], "event: source\n") || !strings.HasPrefix(parts[2], "event: done\n") {
				t.Fatal(body)
			}
			var card sourceResponse
			var reply messageResponse
			if err := json.Unmarshal([]byte(strings.TrimPrefix(parts[1], "event: source\ndata: ")), &card); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(strings.TrimPrefix(parts[2], "event: done\ndata: ")), &reply); err != nil {
				t.Fatal(err)
			}
			if reply.Source == nil || card != *reply.Source || card != *sourceView(source) {
				t.Fatal(body)
			}
		} else if len(parts) != 2 || strings.Contains(body, "event: source") || !strings.Contains(body, `"source":null`) {
			t.Fatal(body)
		}
	}
}

func TestFirstMessageStream(t *testing.T) {
	for _, failure := range []bool{false, true} {
		service := &fakeDialog{send: func(_ context.Context, user users.User, id, text string, emit func(buddy.Event) error) (buddy.Message, error) {
			if user.ID != "user" || id != "" || text != "Первый вопрос" {
				t.Fatal("wrong new chat inputs")
			}
			if err := emit(buddy.Event{Type: "chat", Chat: &buddy.Chat{ID: "created", UserID: user.ID, Title: text}}); err != nil {
				return buddy.Message{}, err
			}
			if failure {
				return buddy.Message{}, errors.New("internal secret")
			}
			if err := emit(buddy.Event{Type: "text", Text: "Ответ"}); err != nil {
				return buddy.Message{}, err
			}
			return buddy.Message{ID: 2, ChatID: "created", Role: buddy.RoleAssistant, Content: "Ответ"}, nil
		}}
		w := messageRequestAt(service, "/api/chats/messages", `{"content":"Первый вопрос"}`, true)
		body := w.Body.String()
		if w.Code != 200 || w.Header().Get("Content-Type") != "text/event-stream" || !w.Flushed || !strings.HasPrefix(body, "event: chat\n") || !strings.Contains(body, `"id":"created","user_id":"user","title":"Первый вопрос"`) {
			t.Fatalf("new chat stream: %d %s", w.Code, body)
		}
		if failure {
			if !strings.Contains(body, "event: error") || strings.Contains(body, "event: done") || strings.Contains(body, "secret") {
				t.Fatal(body)
			}
		} else if !strings.Contains(body, "event: done") || !strings.Contains(body, `"chat_id":"created"`) {
			t.Fatal(body)
		}
	}
}

func TestFirstMessageValidation(t *testing.T) {
	for _, body := range []string{`broken`, `{}`, `{"content":" "}`, `{"content":3}`, `{"content":"q"} {}`} {
		service := &fakeDialog{}
		w := messageRequestAt(service, "/api/chats/messages", body, true)
		if w.Code != 400 || service.calls != 0 {
			t.Fatalf("invalid new chat: %d calls %d", w.Code, service.calls)
		}
	}
	service := &fakeDialog{}
	if w := messageRequestAt(service, "/api/chats/messages", `{"content":"q"}`, false); w.Code != 401 || service.calls != 0 {
		t.Fatalf("unauthorized new chat: %d calls %d", w.Code, service.calls)
	}
}

func (d *fakeDialog) ConfirmTicket(ctx context.Context, user users.User, chatID string, messageID int64, title, description string) (buddy.Ticket, bool, error) {
	d.calls++
	return d.confirm(ctx, user, chatID, messageID, title, description)
}
