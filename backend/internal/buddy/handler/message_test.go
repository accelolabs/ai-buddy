package handler

import (
	"backend/internal/buddy"
	"backend/internal/users"
	"backend/internal/users/middleware"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fakeAgent struct {
	generate func(context.Context, users.User, []buddy.Message, func(buddy.Event) error) (buddy.Answer, error)
	calls    int
}

func (a *fakeAgent) Generate(ctx context.Context, u users.User, h []buddy.Message, emit func(buddy.Event) error) (buddy.Answer, error) {
	a.calls++
	return a.generate(ctx, u, h, emit)
}

type userRepository struct{}

func (userRepository) Get(_ context.Context, id string) (users.User, error) {
	if id != "intern" {
		return users.User{}, sql.ErrNoRows
	}
	return users.User{ID: id, Role: users.RoleIntern}, nil
}

func send(h *Handler, body, user string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/api/buddy/messages", strings.NewReader(body))
	if user != "" {
		r.AddCookie(&http.Cookie{Name: "demo_user", Value: user})
	}
	w := httptest.NewRecorder()
	middleware.Middleware(userRepository{})(http.HandlerFunc(h.Send)).ServeHTTP(w, r)
	return w
}

func TestValidation(t *testing.T) {
	a := &fakeAgent{}
	h := NewHandler(a, 6)
	for _, body := range []string{
		``, `null`, `{}`, `{"messages":null}`, `{"messages":[]}`,
		`{"messages":[{"role":"system","content":"x"}]}`,
		`{"messages":[{"role":"tool","content":"x"}]}`,
		`{"messages":[{"role":"user","content":" "}]}`,
		`{"messages":[{"role":"assistant","content":"x"}]}`,
		`{"messages":[{"role":"user","content":"x"},{"role":"assistant","content":" "},{"role":"user","content":"y"}]}`,
		`{"messages":[{"role":"user","content":"x"}]} {}`,
		`{"messages":[{"role":"user","content":"x"}]} garbage`,
		`{"messages":[{"role":"user","content":"` + strings.Repeat("a", 1<<20) + `"}]}`,
		`{"messages":[{"role":"user","content":"x"}]}` + strings.Repeat(" ", 1<<20),
	} {
		if w := send(h, body, "intern"); w.Code != 400 {
			t.Fatalf("validation: %d %s", w.Code, w.Body)
		}
	}
	for _, u := range []string{"", "missing"} {
		if w := send(h, `{"messages":[{"role":"user","content":"x"}]}`, u); w.Code != 401 {
			t.Fatalf("user: %d", w.Code)
		}
	}
	if a.calls != 0 {
		t.Fatal("invalid request reached agent")
	}
}

func TestContextAndSSE(t *testing.T) {
	a := &fakeAgent{generate: func(_ context.Context, u users.User, h []buddy.Message, emit func(buddy.Event) error) (buddy.Answer, error) {
		if u.ID != "intern" || !reflect.DeepEqual(h, []buddy.Message{{Role: "assistant", Content: "second"}, {Role: "user", Content: " third "}}) {
			t.Fatalf("context: %+v %+v", u, h)
		}
		if err := emit(buddy.Event{Type: "state", State: "thinking"}); err != nil {
			return buddy.Answer{}, err
		}
		if err := emit(buddy.Event{Type: "text", Text: "reply"}); err != nil {
			return buddy.Answer{}, err
		}
		return buddy.Answer{Content: "reply", Source: &buddy.Source{DocumentID: "doc"}, Task: &buddy.Task{Description: "Задача", Deadline: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)}, Ticket: &buddy.Ticket{Title: "title", Description: "description", Recipient: buddy.Recipient{ID: "mentor", Name: "Mentor"}}}, nil
	}}
	w := send(NewHandler(a, 2), `{"messages":[{"role":"user","content":"first"},{"role":"assistant","content":"second"},{"role":"user","content":" third "}]}`, "intern")
	body := w.Body.String()
	if w.Code != 200 || w.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("response: %d %s", w.Code, body)
	}
	last := -1
	for _, event := range []string{"state", "text", "source", "ticket", "task", "done"} {
		pos := strings.Index(body, "event: "+event+"\n")
		if pos <= last {
			t.Fatalf("event order: %s", body)
		}
		last = pos
	}
	if !strings.Contains(body, "event: task\ndata: {\"description\":\"Задача\",\"deadline\":\"2030-01-01T00:00:00Z\"}") || !strings.Contains(body, `"task":{"description":"Задача","deadline":"2030-01-01T00:00:00Z"}`) {
		t.Fatal(body)
	}
	for _, field := range []string{`"status"`, `"ticket_id"`, `"chat_id"`, `"created_at"`, "event: chat"} {
		if strings.Contains(body, field) {
			t.Fatalf("legacy field %s: %s", field, body)
		}
	}
	if !strings.Contains(body, `data: {"content":"reply","source":{"document_id":"doc"`) {
		t.Fatal(body)
	}
}

func TestStreamErrors(t *testing.T) {
	for _, stage := range []string{"before", "after", "cancel", "write", "flush", "unknown"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			a := &fakeAgent{generate: func(_ context.Context, _ users.User, _ []buddy.Message, emit func(buddy.Event) error) (buddy.Answer, error) {
				if stage == "cancel" {
					cancel()
					return buddy.Answer{}, ctx.Err()
				}
				if stage == "unknown" {
					return buddy.Answer{}, emit(buddy.Event{Type: "chat"})
				}
				if stage != "before" {
					if err := emit(buddy.Event{Type: "text", Text: "partial"}); err != nil {
						return buddy.Answer{}, err
					}
				}
				return buddy.Answer{Source: &buddy.Source{}, Ticket: &buddy.Ticket{}, Task: &buddy.Task{}}, errors.New("private details")
			}}
			r := httptest.NewRequest("POST", "/api/buddy/messages", strings.NewReader(`{"messages":[{"role":"user","content":"x"}]}`)).WithContext(ctx)
			r.AddCookie(&http.Cookie{Name: "demo_user", Value: "intern"})
			w := &failingStreamWriter{ResponseRecorder: httptest.NewRecorder()}
			if stage == "write" {
				w.writeErr = errors.New("write")
			}
			if stage == "flush" {
				w.flushErr = errors.New("flush")
			}
			middleware.Middleware(userRepository{})(http.HandlerFunc(NewHandler(a, 6).Send)).ServeHTTP(w, r)
			body := w.Body.String()
			for _, event := range []string{"source", "ticket", "task", "done"} {
				if strings.Contains(body, "event: "+event) {
					t.Fatal(body)
				}
			}
			if strings.Contains(body, "private") {
				t.Fatal(body)
			}
			if (stage == "before" || stage == "unknown") && w.Code != 500 {
				t.Fatalf("before: %d %s", w.Code, body)
			}
			if stage == "after" && !strings.Contains(body, "event: error") {
				t.Fatal(body)
			}
			if stage == "cancel" && body != "" {
				t.Fatal(body)
			}
		})
	}
}

func TestNoAttachmentsOrHistory(t *testing.T) {
	a := &fakeAgent{generate: func(_ context.Context, _ users.User, h []buddy.Message, _ func(buddy.Event) error) (buddy.Answer, error) {
		if len(h) != 1 || h[0].Content != "new" {
			t.Fatal(h)
		}
		return buddy.Answer{Content: "answer"}, nil
	}}
	for range 2 {
		w := send(NewHandler(a, 6), `{"messages":[{"role":"user","content":"new"}]}`, "intern")
		if w.Body.String() != "event: done\ndata: {\"content\":\"answer\",\"source\":null,\"ticket\":null,\"task\":null}\n\n" {
			t.Fatal(w.Body)
		}
	}
}
