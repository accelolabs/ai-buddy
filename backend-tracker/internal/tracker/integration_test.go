package tracker

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"ai-buddy/backend-tracker/internal/database"
	"ai-buddy/backend-tracker/internal/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresIntegration(t *testing.T) {
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
	schema := fmt.Sprintf("backend_tracker_domain_%d", time.Now().UnixNano())
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
	db.SetMaxOpenConns(12)
	for _, command := range []string{"up", "up", "status", "down", "up"} {
		if e = migrations.Run(ctx, db, command); e != nil {
			t.Fatal(command, e)
		}
	}
	for _, q := range []string{
		`INSERT INTO profiles(authentik_id,name,username,email,role,team) VALUES ('i','I','i','i@test','intern','backend'),('j','J','j','j@test','intern','platform'),('m','M','m','m@test','mentor','backend'),('n','N','n','n@test','mentor','backend')`,
		`INSERT INTO assignments(intern_id,mentor_id,team) VALUES ('i','m','backend')`,
	} {
		if _, e = db.ExecContext(ctx, q); e != nil {
			t.Fatal(e)
		}
	}
	for _, q := range []string{
		`INSERT INTO assignments(intern_id,mentor_id,team) VALUES ('i','n','backend')`,
		`INSERT INTO assignments(intern_id,mentor_id,team) VALUES ('j','m','platform')`,
		`INSERT INTO assignments(intern_id,mentor_id,team) VALUES ('m','n','backend')`,
		`INSERT INTO assignments(intern_id,mentor_id,team) VALUES ('j','i','platform')`,
	} {
		if _, e = db.ExecContext(ctx, q); e == nil {
			t.Fatal("invalid assignment accepted", q)
		}
	}
	s := New(&Postgres{DB: db}, time.UTC)
	in := Input{Title: "Task", Description: "Concurrent", Deadline: "2020-01-01"}
	var wg sync.WaitGroup
	ids := make(chan string, 20)
	created := make(chan bool, 20)
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, c, e := s.Create(ctx, "i", "concurrent", in)
			ids <- v.ID
			created <- c
			errs <- e
		}()
	}
	wg.Wait()
	close(ids)
	close(created)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	id := ""
	for v := range ids {
		if id == "" {
			id = v
		}
		if v != id {
			t.Fatal("multiple task IDs")
		}
	}
	count := 0
	for c := range created {
		if c {
			count++
		}
	}
	if count != 1 {
		t.Fatal("created", count)
	}
	ts, e := s.Tasks(ctx, "i")
	if e != nil || len(ts) != 1 || !ts[0].Overdue {
		t.Fatal(ts, e)
	}
	if _, e = s.Complete(ctx, "j", id); e != ErrNotFound {
		t.Fatal("foreign completion", e)
	}
	first, e := s.Complete(ctx, "i", id)
	if e != nil {
		t.Fatal(e)
	}
	second, e := s.Complete(ctx, "i", id)
	if e != nil || first.CompletedAt == nil || !first.CompletedAt.Equal(*second.CompletedAt) {
		t.Fatal("completion persistence", e)
	}
	in.Description = "changed"
	if _, _, e = s.Create(ctx, "i", "concurrent", in); e != ErrConflict {
		t.Fatal("conflict", e)
	}
	p, e := s.MentorProgress(ctx, "m", "i")
	if e != nil || p.Percent != 100 || p.Mentor.ID != "m" {
		t.Fatal(p, e)
	}
	if m, e := s.Mentor(ctx, "j"); e != nil || m != nil {
		t.Fatal(m, e)
	}
}
