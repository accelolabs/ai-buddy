package repository

import (
	"backend/internal/buddy"
	"backend/internal/wiki"
	wikirepository "backend/internal/wiki/repository"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Each test applies the real migrations in a separate schema.
func testRepository(t *testing.T) (*Repository, context.Context) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin := stdlib.OpenDB(*cfg)
	t.Cleanup(func() {
		if err := admin.Close(); err != nil {
			t.Error(err)
		}
	})
	schema := fmt.Sprintf("buddy_test_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := admin.ExecContext(cleanupCtx, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Error(err)
		}
	})

	cfg.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*cfg)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpContext(ctx, db, "../../../migrations"); err != nil {
		t.Fatal(err)
	}

	return NewRepository(db), ctx
}

func TestChatAndMessages(t *testing.T) {
	repo, ctx := testRepository(t)
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO users (id, name, role, team) VALUES ('user', 'User', 'intern', 'backend')`); err != nil {
		t.Fatal(err)
	}
	first, err := repo.AddChat(ctx, buddy.Chat{ID: "a", UserID: "user", Title: "First"})
	if err != nil || first.CreatedAt.IsZero() {
		t.Fatalf("add: %+v, %v", first, err)
	}
	got, err := repo.GetChat(ctx, "a")
	if err != nil || got != first {
		t.Fatalf("get: %+v, %v", got, err)
	}
	if _, err := repo.AddChat(ctx, first); err == nil {
		t.Fatal("duplicate accepted")
	}
	if _, err := repo.AddChat(ctx, buddy.Chat{ID: "bad", UserID: "missing"}); err == nil {
		t.Fatal("unknown owner accepted")
	}
	if _, err := repo.GetChat(ctx, "missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	if _, err := repo.AddChat(ctx, buddy.Chat{ID: "b", UserID: "user"}); err != nil {
		t.Fatal(err)
	}
	empty, err := repo.ListMessages(ctx, "a")
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty: %v, %v", empty, err)
	}
	var saved []buddy.Message
	for _, role := range []string{buddy.RoleUser, buddy.RoleAssistant} {
		message, err := repo.AddMessage(ctx, buddy.Message{ChatID: "a", Role: role, Content: "Текст"})
		if err != nil || message.ID == 0 || message.CreatedAt.IsZero() {
			t.Fatalf("add message: %+v, %v", message, err)
		}
		saved = append(saved, message)
	}
	history, err := repo.ListMessages(ctx, "a")
	if err != nil || len(history) != 2 || history[0] != saved[0] || history[1] != saved[1] || history[0].ID >= history[1].ID {
		t.Fatalf("history: %v, %v", history, err)
	}
	if got, err := repo.ListMessages(ctx, "b"); err != nil || len(got) != 0 {
		t.Fatalf("other chat: %v, %v", got, err)
	}
	for _, message := range []buddy.Message{{ChatID: "missing", Role: buddy.RoleUser}, {ChatID: "a", Role: "tool"}} {
		if _, err := repo.AddMessage(ctx, message); err == nil {
			t.Fatal("invalid message accepted")
		}
	}
}

func TestSourceSnapshot(t *testing.T) {
	repo, ctx := testRepository(t)
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO users (id,name,role,team) VALUES ('user','User','intern','backend')`); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddChat(ctx, buddy.Chat{ID: "chat", UserID: "user"}); err != nil {
		t.Fatal(err)
	}
	wikiRepo := wikirepository.NewRepository(repo.db)
	document := wiki.Document{ID: "doc", Title: "Old title", Path: "doc.md"}
	chunk := wiki.Chunk{DocumentID: "doc", Ordinal: 2, Anchor: "экватор", Text: "Full quote\nSecond line"}
	if err := wikiRepo.Sync(ctx, []wiki.Document{document}, []wiki.Chunk{chunk}); err != nil {
		t.Fatal(err)
	}
	source := &buddy.Source{DocumentID: document.ID, Ordinal: chunk.Ordinal, Title: document.Title, Anchor: chunk.Anchor, Quote: chunk.Text}
	saved, err := repo.AddMessage(ctx, buddy.Message{ChatID: "chat", Role: buddy.RoleAssistant, Content: "Answer", Source: source})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddMessage(ctx, buddy.Message{ChatID: "chat", Role: "invalid", Content: "Failed", Source: source}); err == nil {
		t.Fatal("invalid insert accepted")
	}
	if err := wikiRepo.Sync(ctx, []wiki.Document{{ID: "doc", Title: "New title", Path: "doc.md"}}, []wiki.Chunk{{DocumentID: "doc", Ordinal: 2, Text: "New quote"}}); err != nil {
		t.Fatal(err)
	}
	for _, deleteDocument := range []bool{false, true} {
		if deleteDocument {
			if err := wikiRepo.Sync(ctx, nil, nil); err != nil {
				t.Fatal(err)
			}
		}
		history, err := repo.ListMessages(ctx, "chat")
		if err != nil || len(history) != 1 || !reflect.DeepEqual(history[0], saved) {
			t.Fatalf("snapshot changed: %+v %v", history, err)
		}
	}
	if _, err := repo.AddMessage(ctx, buddy.Message{ChatID: "chat", Role: buddy.RoleUser, Content: "No source"}); err != nil {
		t.Fatal(err)
	}
	history, err := repo.ListMessages(ctx, "chat")
	if err != nil || len(history) != 2 || history[1].Source != nil {
		t.Fatalf("null source: %+v %v", history, err)
	}
	var attachment bool
	if err := repo.db.QueryRowContext(ctx, `SELECT attachment FROM messages WHERE id=$1`, history[1].ID).Scan(&attachment); err != nil || attachment {
		t.Fatal("message without attachments must have attachment=false", err)
	}
}

func TestMessageAttachmentTransaction(t *testing.T) {
	repo, ctx := testRepository(t)
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO users (id,name,role,team) VALUES ('user','User','intern','backend')`); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddChat(ctx, buddy.Chat{ID: "chat", UserID: "user"}); err != nil {
		t.Fatal(err)
	}
	// Force failure after the message has been inserted, while writing its source.
	if _, err := repo.db.ExecContext(ctx, `ALTER TABLE message_sources ADD CHECK (quote <> 'Rejected')`); err != nil {
		t.Fatal(err)
	}
	source := &buddy.Source{DocumentID: "doc", Ordinal: 2, Title: "Title", Anchor: "", Quote: "Rejected"}
	if _, err := repo.AddMessage(ctx, buddy.Message{ChatID: "chat", Role: buddy.RoleAssistant, Content: "Answer", Source: source}); err == nil {
		t.Fatal("source failure accepted")
	}
	for _, query := range []string{`SELECT count(*) FROM messages`, `SELECT count(*) FROM message_sources`} {
		var count int
		if err := repo.db.QueryRowContext(ctx, query).Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial message saved: %d %v", count, err)
		}
	}
	source.Quote = "Full quote"
	saved, err := repo.AddMessage(ctx, buddy.Message{ChatID: "chat", Role: buddy.RoleAssistant, Content: "Answer", Source: source})
	if err != nil {
		t.Fatal(err)
	}
	var attachment bool
	if err := repo.db.QueryRowContext(ctx, `SELECT attachment FROM messages WHERE id=$1`, saved.ID).Scan(&attachment); err != nil || !attachment {
		t.Fatal("source must set attachment=true", err)
	}
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO message_sources SELECT * FROM message_sources WHERE message_id=$1`, saved.ID); err == nil {
		t.Fatal("second source accepted")
	}
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO message_sources (message_id,document_id,ordinal,title,anchor,quote) VALUES (-1,'doc',0,'Title','','Quote')`); err == nil {
		t.Fatal("orphan source accepted")
	}
	if _, err := repo.db.ExecContext(ctx, `DELETE FROM messages WHERE id=$1`, saved.ID); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := repo.db.QueryRowContext(ctx, `SELECT count(*) FROM message_sources`).Scan(&count); err != nil || count != 0 {
		t.Fatal("source not deleted with message", err)
	}
}

func TestAttachmentHistory(t *testing.T) {
	repo, ctx := testRepository(t)
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO users (id,name,role,team) VALUES ('user','User','intern','backend')`); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"chat", "other"} {
		if _, err := repo.AddChat(ctx, buddy.Chat{ID: id, UserID: "user"}); err != nil {
			t.Fatal(err)
		}
	}
	var expected []buddy.Message
	for _, message := range []buddy.Message{
		{ChatID: "chat", Role: buddy.RoleUser, Content: "Question"},
		{ChatID: "chat", Role: buddy.RoleAssistant, Content: "First", Source: &buddy.Source{DocumentID: "doc", Ordinal: 0, Title: "First", Anchor: "экватор", Quote: "First quote"}},
		{ChatID: "chat", Role: buddy.RoleAssistant, Content: "Second", Source: &buddy.Source{DocumentID: "doc", Ordinal: 1, Title: "Second", Anchor: "", Quote: "Second quote"}},
		{ChatID: "other", Role: buddy.RoleAssistant, Content: "Other", Source: &buddy.Source{DocumentID: "other", Title: "Other", Quote: "Other quote"}},
	} {
		saved, err := repo.AddMessage(ctx, message)
		if err != nil {
			t.Fatal(err)
		}
		if message.ChatID == "chat" {
			expected = append(expected, saved)
		}
	}
	// The flag can represent a future attachment kind without a source.
	if _, err := repo.db.ExecContext(ctx, `UPDATE messages SET attachment=true WHERE id=$1`, expected[0].ID); err != nil {
		t.Fatal(err)
	}
	history, err := repo.ListMessages(ctx, "chat")
	if err != nil || !reflect.DeepEqual(history, expected) {
		t.Fatalf("history: %+v %v", history, err)
	}
	// A source is only loaded for messages marked as having attachments.
	if _, err := repo.db.ExecContext(ctx, `UPDATE messages SET attachment=false WHERE id=$1`, expected[1].ID); err != nil {
		t.Fatal(err)
	}
	expected[1].Source = nil
	history, err = repo.ListMessages(ctx, "chat")
	if err != nil || !reflect.DeepEqual(history, expected) {
		t.Fatalf("attachment flag ignored: %+v %v", history, err)
	}
}

func TestChatMigration(t *testing.T) {
	repo, ctx := testRepository(t)
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO users (id,name,role,team) VALUES ('user','User','intern','backend')`); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddChat(ctx, buddy.Chat{ID: "chat", UserID: "user"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddMessage(ctx, buddy.Message{ChatID: "chat", Role: buddy.RoleAssistant, Content: "Answer", Source: &buddy.Source{DocumentID: "doc", Quote: "Quote"}}); err != nil {
		t.Fatal(err)
	}
	if err := goose.DownContext(ctx, repo.db, "../../../migrations"); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"chats", "messages", "message_sources"} {
		var absent bool
		if err := repo.db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NULL`, table).Scan(&absent); err != nil || !absent {
			t.Fatalf("rollback left %s: %v", table, err)
		}
	}
	if err := goose.UpContext(ctx, repo.db, "../../../migrations"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddChat(ctx, buddy.Chat{ID: "chat", UserID: "user"}); err != nil {
		t.Fatal(err)
	}
	source := &buddy.Source{DocumentID: "doc", Ordinal: 2, Title: "Title", Quote: "Quote"}
	saved, err := repo.AddMessage(ctx, buddy.Message{ChatID: "chat", Role: buddy.RoleAssistant, Content: "Answer", Source: source})
	if err != nil {
		t.Fatal(err)
	}
	history, err := repo.ListMessages(ctx, "chat")
	if err != nil || !reflect.DeepEqual(history, []buddy.Message{saved}) {
		t.Fatalf("reapplied migration: %+v %v", history, err)
	}
}

func TestListChats(t *testing.T) {
	repo, ctx := testRepository(t)
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO users (id,name,role,team) VALUES ('user','User','intern','backend'), ('other','Other','intern','backend')`); err != nil {
		t.Fatal(err)
	}
	chats, err := repo.ListChats(ctx, "user")
	if err != nil || chats == nil || len(chats) != 0 {
		t.Fatalf("empty: %+v %v", chats, err)
	}
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO chats (id,user_id,title,created_at) VALUES
	  ('old','user','Old','2026-01-01'), ('a','user','A','2026-01-02'),
	  ('b','user','B','2026-01-02'), ('foreign','other','Foreign','2026-01-03')`); err != nil {
		t.Fatal(err)
	}
	chats, err = repo.ListChats(ctx, "user")
	if err != nil || len(chats) != 3 {
		t.Fatalf("list: %+v %v", chats, err)
	}
	for i, id := range []string{"b", "a", "old"} {
		if chats[i].ID != id || chats[i].UserID != "user" || chats[i].Title == "" || chats[i].CreatedAt.IsZero() {
			t.Fatalf("order/isolation: %+v", chats)
		}
	}
}
