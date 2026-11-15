package wiki

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestParseDocument(t *testing.T) {
	for _, tt := range []struct {
		path, content, team, title string
	}{
		{"guide.md", "# Общая статья\nText\n# Второй заголовок", "", "Общая статья"},
		{"backend/guide.md", "# Backend\r\nText", "backend", "Backend"},
		{"platform/nested/guide.md", "# Platform", "platform", "Platform"},
		{"backend/no-heading.md", "No heading", "backend", "no-heading"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			document := parseDocument(tt.path, tt.content)
			if document.Path != tt.path || document.Team != tt.team || document.Title != tt.title {
				t.Fatalf("document = %+v, want path %q, team %q, title %q", document, tt.path, tt.team, tt.title)
			}
			updated := parseDocument(tt.path, "# Changed")
			if document.ID == "" || document.ID != updated.ID {
				t.Fatal("document ID depends on content")
			}
		})
	}
}

func TestIndex(t *testing.T) {
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
	t.Cleanup(func() { _ = admin.Close() })
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
	t.Cleanup(func() { _ = db.Close() })
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpContext(ctx, db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	root := t.TempDir()
	service := NewService(repo, root)
	write := func(name, content string) {
		t.Helper()
		target := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("guide.md", "# Общая статья")
	write("backend/guide.md", "# Backend")
	write("platform/nested/guide.md", "# Platform")
	write("ignored.txt", "Not Markdown")
	for range 2 {
		if err := service.Index(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM documents`).Scan(&count); err != nil || count != 3 {
		t.Fatalf("document count = %d, %v; want 3", count, err)
	}
	for _, name := range []string{"guide.md", "backend/guide.md", "platform/nested/guide.md"} {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		want := parseDocument(name, string(content))
		var got Document
		err = db.QueryRowContext(ctx, `SELECT id, path, team, title FROM documents WHERE path = $1`, name).
			Scan(&got.ID, &got.Path, &got.Team, &got.Title)
		if err != nil || got != want {
			t.Fatalf("stored document = %+v, %v; want %+v", got, err, want)
		}
	}

	write("guide.md", "# Updated")
	if err := service.Index(ctx); err != nil {
		t.Fatal(err)
	}
	var title string
	if err := db.QueryRowContext(ctx, `SELECT title FROM documents WHERE path = 'guide.md'`).
		Scan(&title); err != nil || title != "Updated" {
		t.Fatalf("updated title = %q, %v", title, err)
	}

	var version string
	if err := db.QueryRowContext(ctx, `SELECT xmin::text FROM documents WHERE path = 'guide.md'`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := service.Index(ctx); err != nil {
		t.Fatal(err)
	}
	var unchangedVersion string
	if err := db.QueryRowContext(ctx, `SELECT xmin::text FROM documents WHERE path = 'guide.md'`).
		Scan(&unchangedVersion); err != nil || unchangedVersion != version {
		t.Fatalf("unchanged document was rewritten: version %q -> %q, %v", version, unchangedVersion, err)
	}

	// A failed insert must roll back both updates and deletion of missing files.
	if _, err := db.ExecContext(ctx, `ALTER TABLE documents ADD CHECK (title <> 'Rejected')`); err != nil {
		t.Fatal(err)
	}
	write("guide.md", "# Uncommitted")
	write("z-failure.md", "# Rejected")
	if err := os.Remove(filepath.Join(root, "backend/guide.md")); err != nil {
		t.Fatal(err)
	}
	if err := service.Index(ctx); err == nil {
		t.Fatal("index unexpectedly succeeded")
	}
	if err := db.QueryRowContext(ctx, `SELECT title FROM documents WHERE path = 'guide.md'`).
		Scan(&title); err != nil || title != "Updated" {
		t.Fatalf("failed index changed title to %q, %v", title, err)
	}
	if err := NewService(repo, filepath.Join(root, "missing")).Index(ctx); err == nil {
		t.Fatal("index accepted missing corpus")
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM documents`).Scan(&count); err != nil || count != 3 {
		t.Fatalf("failed index deleted documents: count = %d, %v; want 3", count, err)
	}

	if err := os.Remove(filepath.Join(root, "z-failure.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, "guide.md"), filepath.Join(root, "platform/renamed.md")); err != nil {
		t.Fatal(err)
	}
	if err := service.Index(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM documents`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("count after deletion and rename = %d, %v; want 2", count, err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM documents WHERE path IN ('guide.md', 'backend/guide.md')`,
	).Scan(&count); err != nil || count != 0 {
		t.Fatalf("obsolete paths remain: count = %d, %v", count, err)
	}
	var renamed Document
	if err := db.QueryRowContext(ctx,
		`SELECT id, path, team, title FROM documents WHERE path = 'platform/renamed.md'`,
	).Scan(&renamed.ID, &renamed.Path, &renamed.Team, &renamed.Title); err != nil {
		t.Fatal(err)
	}
	if want := parseDocument("platform/renamed.md", "# Uncommitted"); renamed != want {
		t.Fatalf("renamed document = %+v, want %+v", renamed, want)
	}

	for _, name := range []string{"platform/renamed.md", "platform/nested/guide.md"} {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(name))); err != nil {
			t.Fatal(err)
		}
	}
	if err := service.Index(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM documents`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("empty corpus did not clear index: count = %d, %v", count, err)
	}
}
