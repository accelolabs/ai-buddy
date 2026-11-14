// Database helpers create isolated schemas for opt-in PostgreSQL integration tests.
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"ai-buddy/internal/platform/database"
	"ai-buddy/internal/platform/migrations"
)

func Database(t *testing.T, files fs.FS) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("buddy_test_%d", time.Now().UnixNano())
	if _, err = admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		if err := admin.Close(); err != nil {
			t.Error(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, err := admin.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE")
		if err != nil {
			t.Error(err)
		}
		if err := admin.Close(); err != nil {
			t.Error(err)
		}
	})
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.RuntimeParams["search_path"] = schema + ",public"
	db := stdlib.OpenDB(*cfg)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	if err = migrations.Run(ctx, db, files, "up"); err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(12)
	return db
}

// Indexing serializes integration fixtures that share Wiki's database-wide lock.
// Production indexes share one database; test schemas alone do not isolate locks.
func Indexing(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = conn.ExecContext(ctx, `SELECT pg_advisory_lock(716918463)`); err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock(716918463)`); err != nil {
			t.Error(err)
		}
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	})
}
