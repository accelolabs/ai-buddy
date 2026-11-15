package repository

import (
	"backend/internal/buddy"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
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
