package repository

import (
	"backend/internal/buddy"
	"backend/internal/buddy/dialog"
	buddyhandler "backend/internal/buddy/handler"
	"backend/internal/users"
	"backend/internal/users/middleware"
	usersrepository "backend/internal/users/repository"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/pressly/goose/v3"
)

func ticketFixture(t *testing.T) (*Repository, context.Context, buddy.Message) {
	t.Helper()
	r, ctx := testRepository(t)
	if err := users.Seed(ctx, r.db); err != nil {
		t.Fatal(err)
	}
	if _, err := r.AddChat(ctx, buddy.Chat{ID: "chat", UserID: "intern.backend.1"}); err != nil {
		t.Fatal(err)
	}
	card := &buddy.Ticket{Title: "draft", Description: "draft text", Recipient: buddy.Recipient{ID: "mentor.backend", Name: "Сергей Дорогов"}}
	m, err := r.AddMessage(ctx, buddy.Message{ChatID: "chat", Role: buddy.RoleAssistant, Content: "Подтвердите", Ticket: card, Source: &buddy.Source{DocumentID: "doc", Title: "Источник", Quote: "Текст"}})
	if err != nil {
		t.Fatal(err)
	}
	return r, ctx, m
}

func ticketCount(t *testing.T, r *Repository, ctx context.Context, want int) {
	t.Helper()
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM tickets`).Scan(&count); err != nil || count != want {
		t.Fatalf("tickets: %d %v", count, err)
	}
}

func assertCard(t *testing.T, r *Repository, ctx context.Context, m buddy.Message) {
	t.Helper()
	history, err := r.ListMessages(ctx, m.ChatID)
	if err != nil || len(history) != 1 || !reflect.DeepEqual(history[0], m) {
		t.Fatalf("history: %+v %v want %+v", history, err, m)
	}
}

func TestTicketStorageAndConfirmation(t *testing.T) {
	r, ctx, m := ticketFixture(t)
	assertCard(t, r, ctx, m)
	recent, err := r.ListRecentMessages(ctx, "chat", 6)
	if err != nil || len(recent) != 1 || recent[0].Ticket != nil || recent[0].Source != nil {
		t.Fatalf("recent: %+v %v", recent, err)
	}
	var attachment bool
	if err := r.db.QueryRowContext(ctx, `SELECT attachment FROM messages WHERE id=$1`, m.ID).Scan(&attachment); err != nil || !attachment {
		t.Fatalf("attachment: %v %v", attachment, err)
	}
	for _, tc := range []struct {
		user, chat string
		id         int64
	}{
		{"intern.backend.2", "chat", m.ID}, {"intern.backend.1", "wrong", m.ID}, {"intern.backend.1", "chat", m.ID + 1},
	} {
		if _, _, err := r.ConfirmTicket(ctx, tc.user, tc.chat, tc.id, "x", "y"); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("foreign card: %v", err)
		}
	}
	ticketCount(t, r, ctx, 0)
	card, created, err := r.ConfirmTicket(ctx, "intern.backend.1", "chat", m.ID, "edited", "edited text")
	if err != nil || !created || card.TicketID == nil || card.Title != "edited" || card.Description != "edited text" || card.Recipient != m.Ticket.Recipient {
		t.Fatalf("confirm: %+v %v %v", card, created, err)
	}
	m.Ticket = &card
	assertCard(t, r, ctx, m)
	var title, description, user string
	if err := r.db.QueryRowContext(ctx, `SELECT title,description,user_id FROM tickets WHERE id=$1`, *card.TicketID).Scan(&title, &description, &user); err != nil || title != card.Title || description != card.Description || user != "intern.backend.1" {
		t.Fatalf("ticket: %s %s %s %v", title, description, user, err)
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM mentorships WHERE intern_id='intern.backend.1'`); err != nil {
		t.Fatal(err)
	}
	again, created, err := r.ConfirmTicket(ctx, "intern.backend.1", "chat", m.ID, "ignored", "ignored")
	if err != nil || created || !reflect.DeepEqual(again, card) {
		t.Fatalf("retry: %+v %v %v", again, created, err)
	}
	ticketCount(t, r, ctx, 1)
	// The ticket FK is unique and cannot reference a missing ticket.
	other, err := r.AddMessage(ctx, buddy.Message{ChatID: "chat", Role: buddy.RoleAssistant, Ticket: &buddy.Ticket{Title: "other"}})
	if err != nil {
		t.Fatal(err)
	}
	history, err := r.ListMessages(ctx, "chat")
	if err != nil || len(history) != 2 || history[1].Ticket == nil || history[1].Source != nil || !reflect.DeepEqual(history[1], other) {
		t.Fatalf("card without source: %+v %v", history, err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE message_tickets SET ticket_id=$1 WHERE message_id=$2`, *card.TicketID, other.ID); err == nil {
		t.Fatal("duplicate link accepted")
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE message_tickets SET ticket_id='missing' WHERE message_id=$1`, other.ID); err == nil {
		t.Fatal("missing ticket link accepted")
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM chats WHERE id='chat'`); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM message_tickets`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("cascade: %d %v", count, err)
	}
	ticketCount(t, r, ctx, 1)
}

func TestTicketConfirmationMentorChanges(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(fmt.Sprint(remove), func(t *testing.T) {
			r, ctx, m := ticketFixture(t)
			query := `UPDATE mentorships SET mentor_id='mentor.platform' WHERE intern_id='intern.backend.1'`
			if remove {
				query = `DELETE FROM mentorships WHERE intern_id='intern.backend.1'`
			}
			if _, err := r.db.ExecContext(ctx, query); err != nil {
				t.Fatal(err)
			}
			if _, _, err := r.ConfirmTicket(ctx, "intern.backend.1", "chat", m.ID, "edited", "edited"); !errors.Is(err, buddy.ErrMentorChanged) {
				t.Fatalf("mentor change: %v", err)
			}
			assertCard(t, r, ctx, m)
			ticketCount(t, r, ctx, 0)
		})
	}
}

func TestConcurrentConfirmations(t *testing.T) {
	r, ctx, m := ticketFixture(t)
	const n = 12
	type result struct {
		card    buddy.Ticket
		created bool
		err     error
	}
	results := make(chan result, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			<-start
			card, created, err := r.ConfirmTicket(ctx, "intern.backend.1", "chat", m.ID, fmt.Sprint(i), fmt.Sprint(i))
			results <- result{card, created, err}
		})
	}
	close(start)
	wg.Wait()
	close(results)
	var want *buddy.Ticket
	createdCount := 0
	for got := range results {
		if got.err != nil {
			t.Fatal(got.err)
		}
		if got.created {
			createdCount++
		}
		if want == nil {
			want = &got.card
		} else if !reflect.DeepEqual(*want, got.card) {
			t.Fatal("concurrent results differ")
		}
	}
	if createdCount != 1 {
		t.Fatalf("created %d times", createdCount)
	}
	ticketCount(t, r, ctx, 1)
	m.Ticket = want
	assertCard(t, r, ctx, m)
}

func TestTicketTransactionsRollback(t *testing.T) {
	r, ctx, m := ticketFixture(t)
	if _, err := r.db.ExecContext(ctx, `ALTER TABLE message_tickets ADD CHECK (title <> 'fail')`); err != nil {
		t.Fatal(err)
	}
	// Ticket INSERT succeeds, but linking fails; neither edit nor ticket may survive.
	if _, _, err := r.ConfirmTicket(ctx, "intern.backend.1", "chat", m.ID, "fail", "edited"); err == nil {
		t.Fatal("expected update failure")
	}
	ticketCount(t, r, ctx, 0)
	assertCard(t, r, ctx, m)
	if _, err := r.db.ExecContext(ctx, `ALTER TABLE tickets ADD CHECK (title <> 'insert fail')`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.ConfirmTicket(ctx, "intern.backend.1", "chat", m.ID, "insert fail", "edited"); err == nil {
		t.Fatal("expected insert failure")
	}
	ticketCount(t, r, ctx, 0)
	assertCard(t, r, ctx, m)
	// Message and source must roll back if saving its draft fails.
	failed := m
	failed.ID = 0
	failed.Ticket = &buddy.Ticket{Title: "fail"}
	if _, err := r.AddMessage(ctx, failed); err == nil {
		t.Fatal("expected card failure")
	}
	assertCard(t, r, ctx, m)
	var sources int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM message_sources`).Scan(&sources); err != nil || sources != 1 {
		t.Fatalf("orphan source: %d %v", sources, err)
	}
	failed.Ticket = &buddy.Ticket{Title: "ok"}
	failed.Role = buddy.RoleUser
	if _, err := r.AddMessage(ctx, failed); err == nil {
		t.Fatal("user draft accepted")
	}
	assertCard(t, r, ctx, m)
	// A source failure must leave neither a card nor a message.
	if _, err := r.db.ExecContext(ctx, `ALTER TABLE message_sources ADD CHECK (document_id <> 'fail')`); err != nil {
		t.Fatal(err)
	}
	failed.Role = buddy.RoleAssistant
	failed.Source = &buddy.Source{DocumentID: "fail"}
	if _, err := r.AddMessage(ctx, failed); err == nil {
		t.Fatal("source failure accepted")
	}
	assertCard(t, r, ctx, m)
}

func TestTicketMigration(t *testing.T) {
	r, ctx, m := ticketFixture(t)
	if _, _, err := r.ConfirmTicket(ctx, "intern.backend.1", "chat", m.ID, "edited", "text"); err != nil {
		t.Fatal(err)
	}
	if err := goose.DownToContext(ctx, r.db, "../../../migrations", 4); err != nil {
		t.Fatal(err)
	}
	var exists bool
	if err := r.db.QueryRowContext(ctx, `SELECT to_regclass('message_tickets') IS NOT NULL`).Scan(&exists); err != nil || exists {
		t.Fatalf("rollback: %v %v", exists, err)
	}
	ticketCount(t, r, ctx, 1)
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM messages`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("lost message: %d %v", count, err)
	}
	if err := goose.UpContext(ctx, r.db, "../../../migrations"); err != nil {
		t.Fatal(err)
	}
	ticketCount(t, r, ctx, 1)
	history, err := r.ListMessages(ctx, "chat")
	if err != nil || len(history) != 1 || history[0].Source == nil || history[0].Ticket != nil {
		t.Fatalf("reapply: %+v %v", history, err)
	}
}

func TestTicketHTTPPersistenceAndAccess(t *testing.T) {
	r, ctx, m := ticketFixture(t)
	h := buddyhandler.NewHandler(dialog.NewService(r, nil, 6))
	selected := middleware.Middleware(usersrepository.NewRepository(r.db))
	mux := http.NewServeMux()
	mux.Handle("POST /api/chats/{chatID}/messages/{messageID}/ticket/confirm", selected(http.HandlerFunc(h.ConfirmTicket)))
	mux.Handle("GET /api/chats/{id}", selected(http.HandlerFunc(h.GetChat)))
	request := func(method, path, user, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if user != "" {
			req.AddCookie(&http.Cookie{Name: "demo_user", Value: user})
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("%s %s %s: %d %s", method, path, user, w.Code, w.Body)
		}
		return w
	}
	path := fmt.Sprintf("/api/chats/chat/messages/%d/ticket/confirm", m.ID)
	body := `{"title":" Edited ","description":" Final text "}`
	for _, user := range []string{"", "missing"} {
		request("POST", path, user, body, 401)
	}
	request("POST", path, "mentor.backend", body, 403)
	request("POST", path, "intern.backend.2", body, 404)
	request("POST", strings.Replace(path, "/chat/", "/missing/", 1), "intern.backend.1", body, 404)
	request("POST", fmt.Sprintf("/api/chats/chat/messages/%d/ticket/confirm", m.ID+1), "intern.backend.1", body, 404)
	if _, err := r.db.ExecContext(ctx, `DELETE FROM mentorships WHERE intern_id='intern.backend.1'`); err != nil {
		t.Fatal(err)
	}
	request("POST", path, "intern.backend.1", body, 409)
	if _, err := r.db.ExecContext(ctx, `INSERT INTO mentorships VALUES ('intern.backend.1','mentor.backend')`); err != nil {
		t.Fatal(err)
	}
	w := request("POST", path, "intern.backend.1", body, 201)
	var card map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &card); err != nil {
		t.Fatal(err)
	}
	if card["status"] != "created" || card["title"] != "Edited" || card["description"] != "Final text" || card["ticket_id"] == nil {
		t.Fatalf("card: %v", card)
	}
	again := request("POST", path, "intern.backend.1", `{"title":"ignored","description":"ignored"}`, 200)
	if again.Body.String() != w.Body.String() {
		t.Fatal("retry changed card")
	}
	// A new service instance restores exactly the same card through history.
	h = buddyhandler.NewHandler(dialog.NewService(NewRepository(r.db), nil, 6))
	mux.Handle("GET /reload/{id}", selected(http.HandlerFunc(h.GetChat)))
	history := request("GET", "/reload/chat", "intern.backend.1", "", 200)
	var view struct {
		Messages []struct {
			Ticket map[string]any `json:"ticket"`
			Source any            `json:"source"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(history.Body.Bytes(), &view); err != nil || len(view.Messages) != 1 || !reflect.DeepEqual(view.Messages[0].Ticket, card) || view.Messages[0].Source == nil {
		t.Fatalf("history: %s %v", history.Body, err)
	}
	for _, user := range []string{"intern.backend.2", "mentor.backend"} {
		w := request("GET", "/api/chats/chat", user, "", 404)
		if strings.Contains(w.Body.String(), "Final text") {
			t.Fatal("data leaked")
		}
	}
	ticketCount(t, r, ctx, 1)
}
