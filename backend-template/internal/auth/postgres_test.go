package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"ai-buddy/backend-template/internal/config"
	"ai-buddy/backend-template/internal/database"
	"ai-buddy/backend-template/internal/migrations"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// Each run owns an isolated schema, including the goose version table.
func TestPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("backend_template_test_%d", time.Now().UnixNano())
	if _, err = admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*cfg)
	defer db.Close()
	for _, command := range []string{"up", "up", "status", "down", "up"} {
		if err = migrations.Run(ctx, db, command); err != nil {
			t.Fatalf("migration %s: %v", command, err)
		}
	}
	for _, name := range []string{"sessions_expiry_idx"} {
		var count int
		if err = db.QueryRowContext(ctx, "SELECT count(*) FROM pg_indexes WHERE schemaname=$1 AND indexname=$2", schema, name).Scan(&count); err != nil || count != 1 {
			t.Fatalf("index %s: %d %v", name, count, err)
		}
	}

	c := config.Config{SessionLifetime: 24 * time.Hour, SessionIdleTimeout: time.Hour}
	s1, stop1 := NewSessions(db, c)
	defer stop1()
	s2, stop2 := NewSessions(db, c)
	defer stop2()
	f := newFixture(t)
	f.a.Sessions = s1
	cookie, state := f.login(t, nil)
	// Complete the login in another instance using the pending data stored by SCS.
	f.a.Sessions = s2
	w := f.callback(cookie, state)
	if w.Code != 302 {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	var current *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "__Host-session" {
			current = cookie
		}
	}
	if current == nil {
		t.Fatal("no session cookie")
	}
	other := *f.a
	other.Sessions = s1
	f.a = &other
	if w = f.request("GET", "/api/auth/me", current); w.Code != 200 {
		t.Fatalf("second instance: %d %s", w.Code, w.Body.String())
	}
	if w = f.request("POST", "/api/auth/logout", current); w.Code != 204 {
		t.Fatal("logout", w.Code)
	}
	f.a.Sessions = s2
	if w = f.request("GET", "/api/auth/me", current); w.Code != 401 {
		t.Fatal("session not deleted")
	}
	cookie, state = f.login(t, nil)
	w = f.callback(cookie, state)
	if w.Code != 302 {
		t.Fatal("second login", w.Code)
	}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "__Host-session" {
			current = cookie
		}
	}
	if _, err = db.ExecContext(ctx, "UPDATE sessions SET expiry=now()-interval '1 second'"); err != nil {
		t.Fatal(err)
	}
	f.a.Sessions = s2
	if w = f.request("GET", "/api/auth/me", current); w.Code != 401 {
		t.Fatal("expired session accepted")
	}
	// SCS persists the shorter idle deadline, never extending the absolute lifetime.
	loaded, err := s1.Load(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	s1.Put(loaded, "test", "value")
	_, _, err = s1.Commit(loaded)
	if err != nil {
		t.Fatal(err)
	}
	var remaining float64
	if err = db.QueryRowContext(ctx, "SELECT EXTRACT(EPOCH FROM max(expiry)-now()) FROM sessions").Scan(&remaining); err != nil || remaining > 3601 || remaining < 3590 {
		t.Fatalf("idle expiry: %f %v", remaining, err)
	}

}
