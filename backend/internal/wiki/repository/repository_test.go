package wiki

import (
	"backend/internal/wiki"
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
	schema := fmt.Sprintf("wiki_test_%d", time.Now().UnixNano())
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

func TestDocuments(t *testing.T) {
	repo, ctx := testRepository(t)
	list, err := repo.List(ctx)
	if err != nil || list == nil || len(list) != 0 {
		t.Fatalf("empty list: %v, %v", list, err)
	}
	common := wiki.Document{ID: "common", Path: "guide.md", Title: "Общая"}
	backend := wiki.Document{ID: "backend", Path: "backend/guide.md", Team: "backend", Title: "Backend"}
	platform := wiki.Document{ID: "platform", Path: "platform/guide.md", Team: "platform", Title: "Platform"}
	if err := repo.Sync(ctx, []wiki.Document{platform, common, backend}, nil); err != nil {
		t.Fatal(err)
	}
	list, err = repo.List(ctx)
	if err != nil || !reflect.DeepEqual(list, []wiki.Document{backend, common, platform}) {
		t.Fatalf("List(): %v, %v", list, err)
	}
	for _, doc := range []wiki.Document{common, backend, platform} {
		got, err := repo.Get(ctx, doc.ID)
		if err != nil || got != doc {
			t.Fatalf("Get(%q): %v, %v", doc.ID, got, err)
		}
	}
	if _, err := repo.Get(ctx, "missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing: %v", err)
	}
}

func TestChunksSync(t *testing.T) {
	repo, ctx := testRepository(t)
	first := wiki.Document{ID: "a", Path: "a.md", Title: "First"}
	second := wiki.Document{ID: "b", Path: "b.md", Title: "Second"}
	chunks := []wiki.Chunk{
		{DocumentID: "a", Ordinal: 1, Anchor: "раздел", Text: "Second chunk"},
		{DocumentID: "b", Ordinal: 0, Text: "Another document"},
		{DocumentID: "a", Ordinal: 0, Text: "First chunk"},
	}
	for range 2 {
		if err := repo.Sync(ctx, []wiki.Document{first, second}, chunks); err != nil {
			t.Fatal(err)
		}
		got, err := repo.ListChunks(ctx, "a")
		if err != nil || !reflect.DeepEqual(got, []wiki.Chunk{chunks[2], chunks[0]}) {
			t.Fatalf("ListChunks: %#v, %v", got, err)
		}
	}
	if got, err := repo.ListChunks(ctx, "missing"); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("missing chunks: %v, %v", got, err)
	}
	changed := first
	changed.Title = "Changed"
	// Failed inserts must roll back metadata updates, document deletions and chunk replacement.
	for _, bad := range [][]wiki.Chunk{
		{{DocumentID: "missing", Ordinal: 0, Text: "orphan"}},
		{{DocumentID: "a", Ordinal: 0, Text: "one"}, {DocumentID: "a", Ordinal: 0, Text: "duplicate"}},
	} {
		if err := repo.Sync(ctx, []wiki.Document{changed}, bad); err == nil {
			t.Fatal("invalid chunks accepted")
		}
		got, err := repo.Get(ctx, "a")
		if err != nil || got != first {
			t.Fatalf("metadata rollback: %v, %v", got, err)
		}
		if _, err := repo.Get(ctx, "b"); err != nil {
			t.Fatalf("document deletion not rolled back: %v", err)
		}
		stored, err := repo.ListChunks(ctx, "a")
		if err != nil || !reflect.DeepEqual(stored, []wiki.Chunk{chunks[2], chunks[0]}) {
			t.Fatalf("chunks rollback: %#v, %v", stored, err)
		}
	}
	updated := []wiki.Chunk{{DocumentID: "a", Ordinal: 0, Anchor: "new-anchor", Text: "Changed content"}}
	if err := repo.Sync(ctx, []wiki.Document{first}, updated); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.ListChunks(ctx, "a")
	if err != nil || !reflect.DeepEqual(stored, updated) {
		t.Fatalf("updated chunks: %#v, %v", stored, err)
	}
	if stored, err := repo.ListChunks(ctx, "b"); err != nil || len(stored) != 0 {
		t.Fatalf("deleted document chunks: %#v, %v", stored, err)
	}
	if _, err := repo.db.ExecContext(ctx, `DELETE FROM documents WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	if stored, err := repo.ListChunks(ctx, "a"); err != nil || len(stored) != 0 {
		t.Fatalf("cascade: %#v, %v", stored, err)
	}
	if err := repo.Sync(ctx, []wiki.Document{first}, updated); err != nil {
		t.Fatal(err)
	}
	if err := repo.Sync(ctx, nil, nil); err != nil {
		t.Fatal(err)
	}
	if stored, err := repo.ListChunks(ctx, "a"); err != nil || len(stored) != 0 {
		t.Fatalf("empty corpus: %#v, %v", stored, err)
	}
}
