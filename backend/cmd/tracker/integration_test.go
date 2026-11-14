package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"ai-buddy/internal/testutil"
	"ai-buddy/internal/tracker/apperror"
	"ai-buddy/internal/tracker/interns"
	internspostgres "ai-buddy/internal/tracker/interns/postgres"
	"ai-buddy/internal/tracker/migrations"
	"ai-buddy/internal/tracker/progress"
	"ai-buddy/internal/tracker/tasks"
	taskspostgres "ai-buddy/internal/tracker/tasks/postgres"
)

func TestPostgresIntegration(t *testing.T) {
	ctx := context.Background()
	db := testutil.Database(t, migrations.Files)
	var e error

	for _, command := range []string{
		"up",
		"up",
		"status",
		"down",
		"up",
	} {
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

	internRepository := internspostgres.New(db)
	taskRepository := taskspostgres.New(db)
	internService := interns.New(internRepository)
	s := tasks.New(taskRepository, internService, time.UTC, time.Now)
	progressService := progress.New(internRepository, taskRepository, time.UTC, time.Now)
	in := tasks.CreateInput{
		Title:       "Task",
		Description: "Concurrent",
		Deadline:    "2020-01-01",
	}
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

	if _, e = s.Complete(ctx, "j", id); e != apperror.ErrNotFound {
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
	if _, _, e = s.Create(ctx, "i", "concurrent", in); e != apperror.ErrConflict {
		t.Fatal("conflict", e)
	}

	p, e := progressService.MentorProgress(ctx, "m", "i")
	if e != nil || p.Percent != 100 || p.Mentor.ID != "m" {
		t.Fatal(p, e)
	}

	if m, e := internService.Mentor(ctx, "j"); e != nil || m != nil {
		t.Fatal(m, e)
	}
}
