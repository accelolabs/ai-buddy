package seed

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"ai-buddy/backend-tracker/internal/database"
	"ai-buddy/backend-tracker/internal/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresSeedIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	admin, e := database.Open(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()

	schema := fmt.Sprintf("backend_tracker_seed_%d", time.Now().UnixNano())
	if _, e = admin.ExecContext(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	defer func() {
		if _, e := admin.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); e != nil {
			t.Error(e)
		}
	}()

	cfg, e := pgx.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}

	cfg.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*cfg)
	defer db.Close()

	if e = migrations.Run(ctx, db, "up"); e != nil {
		t.Fatal(e)
	}

	srv := api(t, "")
	defer srv.Close()

	c := Config{srv.URL, "token"}
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	run := func() {
		t.Helper()
		if e := RunWithClient(ctx, db, c, srv.Client(), now); e != nil {
			t.Fatal(e)
		}
	}
	run()
	counts := func(profiles, assignments, tasks int) {
		t.Helper()
		for table, want := range map[string]int{
			"profiles":    profiles,
			"assignments": assignments,
			"tasks":       tasks,
		} {
			var n int
			if e := db.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&n); e != nil || n != want {
				t.Fatalf("%s: %d want %d: %v", table, n, want, e)
			}
		}
	}
	counts(6, 3, 12)
	for _, q := range []string{
		`UPDATE profiles SET goals='Custom goals' WHERE authentik_id='2'`,
		`UPDATE tasks SET completed_at='2026-11-14 10:00:00+00' WHERE id='demo:intern.backend.1:1'`,
		`INSERT INTO profiles(authentik_id,name,username,email,role,team) VALUES ('extra','Extra','extra','extra@test','mentor','platform')`,
		`UPDATE assignments SET mentor_id='extra' WHERE intern_id='5'`,
		`INSERT INTO tasks(id,intern_id,team,title,deadline,idempotency_key) VALUES ('custom','2','backend','User task','2026-12-01','custom')`,
	} {
		if _, e = db.ExecContext(ctx, q); e != nil {
			t.Fatal(e)
		}
	}
	// A later seed must not move deadlines, reset completions, or replace profiles.
	now = now.AddDate(0, 0, 30)
	run()
	counts(7, 3, 13)
	var goals, deadline, mentor string
	var completed time.Time
	if e = db.QueryRowContext(ctx, `SELECT goals FROM profiles WHERE authentik_id='2'`).Scan(&goals); e != nil || goals != "Custom goals" {
		t.Fatal(goals, e)
	}

	if e = db.QueryRowContext(
		ctx,
		`SELECT to_char(deadline,'YYYY-MM-DD'),completed_at FROM tasks WHERE id='demo:intern.backend.1:1'`,
	).Scan(
		&deadline,
		&completed,
	); e != nil ||
		deadline != "2026-11-13" ||
		!completed.Equal(time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)) {
		t.Fatal(deadline, completed, e)
	}

	if e = db.QueryRowContext(ctx, `SELECT mentor_id FROM assignments WHERE intern_id='5'`).Scan(&mentor); e != nil || mentor != "extra" {
		t.Fatal(mentor, e)
	}
	// Force a failure after inserting profiles, proving PostgreSQL rolls them back.
	if _, e = db.ExecContext(
		ctx,
		`TRUNCATE tasks,assignments,profiles; ALTER TABLE tasks ADD CONSTRAINT reject_demo CHECK (title='never')`,
	); e != nil {
		t.Fatal(e)
	}

	if e = RunWithClient(ctx, db, c, srv.Client(), now); e == nil {
		t.Fatal("expected constraint failure")
	}

	counts(0, 0, 0)
}
