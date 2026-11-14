package auth

import (
	"context"
	"net/http"
	"testing"
	"testing/fstest"
	"time"

	"ai-buddy/internal/platform/config"
	"ai-buddy/internal/platform/migrations"
	"ai-buddy/internal/testutil"
)

// Each run owns an isolated schema, including the goose version table.
func TestPostgres(t *testing.T) {
	ctx := context.Background()
	db := testutil.Database(t, fstest.MapFS{"001_sessions.sql": &fstest.MapFile{Data: []byte(sessionSQL)}})
	var schema string
	if err := db.QueryRowContext(ctx, "SELECT current_schema()").Scan(&schema); err != nil {
		t.Fatal(err)
	}
	var err error

	for _, command := range []string{
		"up",
		"up",
		"status",
		"down",
		"up",
	} {
		if err = migrations.Run(ctx, db, fstest.MapFS{"001_sessions.sql": &fstest.MapFile{Data: []byte(sessionSQL)}}, command); err != nil {
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

const sessionSQL = `-- +goose Up
CREATE TABLE sessions (token TEXT PRIMARY KEY, data BYTEA NOT NULL, expiry TIMESTAMPTZ NOT NULL);
CREATE INDEX sessions_expiry_idx ON sessions (expiry);

-- +goose Down
DROP TABLE sessions;
`
