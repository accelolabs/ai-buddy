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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type ticketUserStore struct{ role string }

func (s ticketUserStore) Get(context.Context, string) (users.User, error) {
	return users.User{ID: "intern", Role: s.role}, nil
}

func confirmRequest(service *fakeDialog, role, id, body string, selected bool) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	var handler http.Handler = http.HandlerFunc(NewHandler(service).ConfirmTicket)
	if selected {
		handler = middleware.Middleware(ticketUserStore{role})(handler)
	}
	mux.Handle("POST /api/chats/{chatID}/messages/{messageID}/ticket/confirm", handler)
	r := httptest.NewRequest("POST", "/api/chats/chat/messages/"+id+"/ticket/confirm", strings.NewReader(body))
	r.AddCookie(&http.Cookie{Name: "demo_user", Value: "intern"})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestConfirmValidation(t *testing.T) {
	for i, body := range []string{
		"", "null", "[]", "{", `{"title":"x"}`, `{"description":"x"}`, `{"title":" \t\n","description":"x"}`,
		`{"title":"x","description":"\u2003"}`, `{"title":1,"description":"x"}`, `{"title":"x","description":null}`,
		`{"title":"x","description":"y"} {}`, `{"title":"x","description":"y"} junk`,
		`{"title":"` + strings.Repeat("x", 1<<20) + `","description":"y"}`,
		`{"title":"x","description":"y"}` + strings.Repeat(" ", 1<<20),
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			d := &fakeDialog{}
			w := confirmRequest(d, users.RoleIntern, "42", body, true)
			if w.Code != 400 || d.calls != 0 {
				t.Fatalf("validation: %d calls=%d", w.Code, d.calls)
			}
		})
	}
	for _, id := range []string{"bad", "0", "-1", "9223372036854775808"} {
		d := &fakeDialog{}
		w := confirmRequest(d, users.RoleIntern, id, `{"title":"x","description":"y"}`, true)
		if w.Code != 400 || d.calls != 0 {
			t.Fatalf("id: %d", w.Code)
		}
	}
}

func TestConfirmResponseAndErrors(t *testing.T) {
	for _, tc := range []struct {
		name, role        string
		selected, created bool
		err               error
		status            int
	}{
		{name: "created", role: users.RoleIntern, selected: true, created: true, status: 201},
		{name: "retry", role: users.RoleIntern, selected: true, status: 200},
		{name: "unselected", role: users.RoleIntern, status: 401},
		{name: "mentor", role: users.RoleMentor, selected: true, status: 403},
		{name: "missing", role: users.RoleIntern, selected: true, err: dialog.ErrNotFound, status: 404},
		{name: "changed", role: users.RoleIntern, selected: true, err: buddy.ErrMentorChanged, status: 409},
		{name: "internal", role: users.RoleIntern, selected: true, err: errors.New("private details"), status: 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := "ticket-id"
			d := &fakeDialog{confirm: func(_ context.Context, user users.User, chatID string, messageID int64, title, description string) (buddy.Ticket, bool, error) {
				if user.ID != "intern" || chatID != "chat" || messageID != 42 || title != "Тема" || description != "Описание" {
					t.Fatal("wrong arguments")
				}
				return buddy.Ticket{Title: title, Description: description, Recipient: buddy.Recipient{ID: "mentor", Name: "Ментор"}, TicketID: &id}, tc.created, tc.err
			}}
			w := confirmRequest(d, tc.role, "42", `{"title":" Тема ","description":"\nОписание\t"}`, tc.selected)
			if w.Code != tc.status || strings.Contains(w.Body.String(), "private") {
				t.Fatalf("response: %d %s", w.Code, w.Body)
			}
			if w.Code == 200 || w.Code == 201 {
				var card ticketResponse
				if err := json.Unmarshal(w.Body.Bytes(), &card); err != nil {
					t.Fatal(err)
				}
				if card.Status != "created" || card.TicketID == nil || *card.TicketID != id || card.Recipient.Name != "Ментор" {
					t.Fatalf("card: %+v", card)
				}
			}
			if (tc.status == 401 || tc.status == 403) && d.calls != 0 {
				t.Fatal("unauthorized call")
			}
		})
	}
}

func TestTicketSSEAndHistoryContract(t *testing.T) {
	card := &buddy.Ticket{Title: "Тема", Description: "Текст", Recipient: buddy.Recipient{ID: "mentor", Name: "Ментор"}}
	source := &buddy.Source{DocumentID: "doc"}
	reply := buddy.Message{ID: 2, ChatID: "chat", Role: buddy.RoleAssistant, Content: "Ответ", Ticket: card, Source: source}
	d := &fakeDialog{send: func(_ context.Context, _ users.User, _, _ string, emit func(buddy.Event) error) (buddy.Message, error) {
		if err := emit(buddy.Event{Type: "source", Source: source}); err != nil {
			return buddy.Message{}, err
		}
		if err := emit(buddy.Event{Type: "ticket", Ticket: card}); err != nil {
			return buddy.Message{}, err
		}
		return reply, nil
	}}
	w := request(t, d, `{"content":"question"}`, true)
	text := w.Body.String()
	if w.Code != 200 || !strings.Contains(text, `"ticket_id":null`) || !strings.Contains(text, `"status":"draft"`) {
		t.Fatalf("stream: %s", text)
	}
	sourceAt, ticketAt, doneAt := strings.Index(text, "event: source"), strings.Index(text, "event: ticket"), strings.Index(text, "event: done")
	if sourceAt < 0 || ticketAt < sourceAt || doneAt < ticketAt {
		t.Fatal("wrong event order")
	}
	encoded, _ := json.Marshal(messageView(reply))
	if !strings.Contains(text, string(encoded)) {
		t.Fatal("done differs from history format")
	}
	id := "created"
	card.TicketID = &id
	encoded, _ = json.Marshal(messageView(reply))
	if !strings.Contains(string(encoded), `"status":"created"`) {
		t.Fatal("created status")
	}
	encoded, _ = json.Marshal(messageView(buddy.Message{}))
	if !strings.Contains(string(encoded), `"ticket":null`) {
		t.Fatal("missing nullable ticket")
	}
}
