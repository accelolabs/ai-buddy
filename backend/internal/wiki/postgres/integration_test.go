package postgres

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ai-buddy/internal/platform/migrations"
	"ai-buddy/internal/testutil"
	"ai-buddy/internal/wiki"
	"ai-buddy/internal/wiki/indexer"
	wikimigrations "ai-buddy/internal/wiki/migrations"
)

// Run with TEST_DATABASE_URL against PostgreSQL 16 with pgvector installed.
// The helper creates a unique schema; the role must be allowed to create extensions.
func TestPostgresLifecycle(t *testing.T) {
	db := testutil.Database(t, wikimigrations.Files)
	r := &Repository{db}
	ctx := context.Background()
	root := corpus(t, map[string]string{"common/start.md": "# Общее\n\nобщий документ", "backend/private.md": "# Backend\n\nсекрет бэкенда", "platform/private.md": "# Platform\n\nсекрет платформы"})
	e := &fakeEmbedder{fp: "test-model-v1"}
	stats, err := indexer.Run(ctx, r, root, e, false)
	if err != nil || stats.Added != 3 {
		t.Fatal(stats, err)
	}
	session, err := r.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if fp, err := session.Fingerprint(ctx); err != nil || fp != e.fp {
		t.Fatal(fp, err)
	}
	session.Close()
	var dim int
	if err := db.QueryRow("SELECT vector_dims(embedding) FROM chunks LIMIT 1").Scan(&dim); err != nil || dim != 768 {
		t.Fatal(dim, err)
	}
	for _, team := range []string{"backend", "platform"} {
		docs, err := r.List(ctx, team)
		if err != nil || len(docs) != 2 {
			t.Fatal(team, docs, err)
		}
		other := "platform"
		if team == "platform" {
			other = "backend"
		}
		foreign := indexer.Parse(other+"/private.md", "").ID
		if _, err := r.Get(ctx, team, foreign); !errors.Is(err, wiki.ErrNotFound) {
			t.Fatal("foreign document", err)
		}
		results, err := r.Search(ctx, team, "секрет", embedding(), 0.35, e.fp)
		if err != nil || len(results) != 2 {
			t.Fatal(results, err)
		}
		for _, result := range results {
			if result.DocumentID == foreign {
				t.Fatal("search leaked", team)
			}
		}
		own, err := r.Get(ctx, team, indexer.Parse(team+"/private.md", "").ID)
		if err != nil || !strings.Contains(own.Markdown, "секрет") {
			t.Fatal(own, err)
		}
	}
	// Lexical branch finds Russian morphology even with a dissimilar vector.
	v := embedding()
	v[0] = -1
	results, err := r.Search(ctx, "backend", "бэкенда", v, 0.99, e.fp)
	if err != nil || len(results) != 1 {
		t.Fatal(results, err)
	}
	results, err = r.Search(ctx, "backend", "несуществующийтермин", v, 0.99, e.fp)
	if err != nil || len(results) != 0 {
		t.Fatal(results, err)
	}
	if _, err = r.Search(ctx, "backend", "секрет", v, 0.35, "wrong"); !errors.Is(err, wiki.ErrUnavailable) {
		t.Fatal(err)
	}
	before := e.calls
	stats, err = indexer.Run(ctx, r, root, e, false)
	if err != nil || stats.Skipped != 3 || e.calls != before {
		t.Fatal(stats, e.calls, err)
	}
	// A held lock rejects a second connection before accessing files or Ollama.
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Run(ctx, r, root, e, false); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", lockID); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Error(err)
	}
	// Preparation failure preserves the old snapshot, including a pending deletion.
	if err := os.Remove(filepath.Join(root, "common/start.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "backend/private.md"), []byte("# Changed\n\nновый текст"), 0644); err != nil {
		t.Fatal(err)
	}
	e.hook = func() { e.err = wiki.ErrUnavailable }
	if _, err := indexer.Run(ctx, r, root, e, false); err == nil {
		t.Fatal("failed embed succeeded")
	}
	e.hook = nil
	e.err = nil
	if _, err := r.Get(ctx, "backend", indexer.Parse("common/start.md", "").ID); err != nil {
		t.Fatal("failed run removed document", err)
	}
	// Publication failure also rolls back deletions and replacements.
	if _, err := db.Exec("ALTER TABLE chunks ADD CONSTRAINT reject_new CHECK(text <> 'новый текст')"); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Run(ctx, r, root, e, false); err == nil {
		t.Fatal("constraint failure succeeded")
	}
	if _, err := r.Get(ctx, "backend", indexer.Parse("common/start.md", "").ID); err != nil {
		t.Fatal("transaction leaked deletion", err)
	}
	if _, err := db.Exec("ALTER TABLE chunks DROP CONSTRAINT reject_new"); err != nil {
		t.Fatal(err)
	}
	stats, err = indexer.Run(ctx, r, root, e, false)
	if err != nil || stats.Changed != 1 || stats.Deleted != 1 {
		t.Fatal(stats, err)
	}
	// Moving across team directories changes identity and revokes old access.
	if err := os.Rename(filepath.Join(root, "backend/private.md"), filepath.Join(root, "platform/moved.md")); err != nil {
		t.Fatal(err)
	}
	stats, err = indexer.Run(ctx, r, root, e, false)
	if err != nil || stats.Added != 1 || stats.Deleted != 1 {
		t.Fatal(stats, err)
	}
	if _, err := r.Get(ctx, "backend", indexer.Parse("platform/moved.md", "").ID); !errors.Is(err, wiki.ErrNotFound) {
		t.Fatal("move leaked", err)
	}
	e.fp = "test-model-v2"
	if _, err := indexer.Run(ctx, r, root, e, false); err == nil {
		t.Fatal("changed fingerprint accepted")
	}
	before = e.calls
	stats, err = indexer.Run(ctx, r, root, e, true)
	if err != nil || stats.Changed != 2 || e.calls <= before {
		t.Fatal(stats, err)
	}
	// Exact vector dimensionality is enforced in PostgreSQL too.
	if _, err := db.Exec("UPDATE chunks SET embedding='[1,2,3]'::vector"); err == nil {
		t.Fatal("wrong vector dimension accepted")
	}
	if err := migrations.Run(ctx, db, wikimigrations.Files, "down"); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Run(ctx, db, wikimigrations.Files, "up"); err != nil {
		t.Fatal(err)
	}
}
