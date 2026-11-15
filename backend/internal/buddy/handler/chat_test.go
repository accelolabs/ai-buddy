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

func readRequest(service *fakeDialog, target string, auth bool) *httptest.ResponseRecorder {
	h := NewHandler(service)
	mux := http.NewServeMux()
	selected := middleware.Middleware(userStore{})
	mux.Handle("GET /api/chats", selected(http.HandlerFunc(h.ListChats)))
	mux.Handle("GET /api/chats/{id}", selected(http.HandlerFunc(h.GetChat)))
	r := httptest.NewRequest(http.MethodGet, target, nil)
	if auth {
		r.AddCookie(&http.Cookie{Name: "demo_user", Value: "user"})
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestListChats(t *testing.T) {
	for _, empty := range []bool{false, true} {
		service := &fakeDialog{list: func(_ context.Context, user users.User) ([]buddy.Chat, error) {
			if user.ID != "user" {
				t.Fatal("untrusted user")
			}
			if empty {
				return nil, nil
			}
			return []buddy.Chat{{ID: "chat", UserID: user.ID, Title: "Название"}}, nil
		}}
		w := readRequest(service, "/api/chats", true)
		if w.Code != 200 || w.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("response: %d %s", w.Code, w.Body.String())
		}
		var chats []chatResponse
		if err := json.Unmarshal(w.Body.Bytes(), &chats); err != nil {
			t.Fatal(err)
		}
		if empty {
			if w.Body.String() != "[]\n" {
				t.Fatal(w.Body.String())
			}
		} else if len(chats) != 1 || chats[0].ID != "chat" || chats[0].Title != "Название" || chats[0].UserID != "user" {
			t.Fatal(chats)
		}
	}
}

func TestGetChatHistory(t *testing.T) {
	for _, empty := range []bool{false, true} {
		service := &fakeDialog{get: func(_ context.Context, user users.User, id string) (buddy.Chat, []buddy.Message, error) {
			if user.ID != "user" || id != "chat" {
				t.Fatal("wrong chat inputs")
			}
			var messages []buddy.Message
			if !empty {
				// History returned over HTTP is complete, independent of the model's limit.
				for i := range 8 {
					messages = append(messages, buddy.Message{ID: int64(i + 1), ChatID: id, Role: buddy.RoleUser, Content: "Question"})
				}
				messages[7].Role = buddy.RoleAssistant
				messages[7].Source = &buddy.Source{DocumentID: "doc", Ordinal: 2, Title: "Title", Anchor: "section", Quote: "Quote"}
			}
			return buddy.Chat{ID: id, UserID: user.ID, Title: "Title"}, messages, nil
		}}
		w := readRequest(service, "/api/chats/chat", true)
		var chat chatHistoryResponse
		if err := json.Unmarshal(w.Body.Bytes(), &chat); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || w.Header().Get("Content-Type") != "application/json" || chat.ID != "chat" || chat.UserID != "user" || chat.Title != "Title" || chat.Messages == nil {
			t.Fatalf("response: %d %s", w.Code, w.Body.String())
		}
		if empty {
			if len(chat.Messages) != 0 {
				t.Fatal(chat.Messages)
			}
		} else if len(chat.Messages) != 8 || chat.Messages[0].ID != 1 || chat.Messages[0].Source != nil || chat.Messages[7].Source == nil || *chat.Messages[7].Source != (sourceResponse{DocumentID: "doc", Ordinal: 2, Title: "Title", Anchor: "section", Quote: "Quote"}) {
			t.Fatal(chat.Messages)
		}
	}
}

func TestReadChatErrors(t *testing.T) {
	for _, target := range []string{"/api/chats", "/api/chats/chat"} {
		service := &fakeDialog{}
		if w := readRequest(service, target, false); w.Code != 401 || service.calls != 0 {
			t.Fatalf("unauthorized: %d calls %d", w.Code, service.calls)
		}
		for _, failure := range []error{dialog.ErrNotFound, errors.New("database secret")} {
			service := &fakeDialog{
				list: func(context.Context, users.User) ([]buddy.Chat, error) { return nil, failure },
				get: func(context.Context, users.User, string) (buddy.Chat, []buddy.Message, error) {
					return buddy.Chat{}, nil, failure
				},
			}
			w := readRequest(service, target, true)
			want := 500
			if target == "/api/chats/chat" && errors.Is(failure, dialog.ErrNotFound) {
				want = 404
			}
			if w.Code != want || strings.Contains(w.Body.String(), "secret") {
				t.Fatalf("response: %d %s", w.Code, w.Body.String())
			}
		}
	}
}
