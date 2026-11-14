package postgres

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ai-buddy/internal/migrations"
	platformmigrations "ai-buddy/internal/platform/migrations"
	"ai-buddy/internal/testutil"
	"ai-buddy/internal/tracker/apperror"
	"ai-buddy/internal/tracker/progress"
	"ai-buddy/internal/tracker/tasks"
)

func TestTaskIntegration(t *testing.T) {
	db := testutil.Database(t, migrations.Files)
	ctx := context.Background()
	for _, command := range []string{"status", "down", "up"} {
		if err := platformmigrations.Run(ctx, db, migrations.Files, command); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO profiles(id,name,username,email,role,team) VALUES ('i','Intern','i','i@test','intern','backend'),('j','Other','j','j@test','intern','backend'),('m','Mentor','m','m@test','mentor','backend'); INSERT INTO assignments(intern_id,mentor_id,team) VALUES ('i','m','backend')`); err != nil {
		t.Fatal(err)
	}
	repo := New(db)
	loc, _ := time.LoadLocation("Europe/Moscow")
	service := tasks.New(repo, repo, loc, func() time.Time { return time.Date(2026, 11, 14, 21, 0, 0, 0, time.UTC) })
	input := tasks.CreateInput{Title: " Task ", Description: " Details ", Deadline: " 2026-11-14 "}
	var wg sync.WaitGroup
	var winners atomic.Int32
	for range 16 {
		wg.Go(func() {
			task, created, err := service.Create(ctx, "i", "shared", input)
			if err != nil || task.Title != "Task" || !task.Overdue {
				t.Errorf("task=%+v err=%v", task, err)
			}
			if created {
				winners.Add(1)
			}
		})
	}
	wg.Wait()
	if winners.Load() != 1 {
		t.Fatal("winners", winners.Load())
	}
	list, err := service.Tasks(ctx, "i")
	if err != nil || len(list) != 1 {
		t.Fatal(list, err)
	}
	id := list[0].ID
	if _, _, err = service.Create(ctx, "i", "shared", tasks.CreateInput{Title: "Changed", Deadline: "2026-11-14"}); !errors.Is(err, apperror.ErrConflict) {
		t.Fatal(err)
	}
	for _, owner := range []string{"missing", "m"} {
		if _, _, err = service.Create(ctx, owner, "new", input); !errors.Is(err, apperror.ErrNotFound) {
			t.Fatal(owner, err)
		}
		if _, err = service.Tasks(ctx, owner); !errors.Is(err, apperror.ErrNotFound) {
			t.Fatal(owner, err)
		}
	}
	if _, err = service.Complete(ctx, "j", id); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatal(err)
	}
	done, err := service.Complete(ctx, "i", id)
	if err != nil || done.CompletedAt == nil || done.Overdue || done.Status != "completed" {
		t.Fatal(done, err)
	}
	again, err := service.Complete(ctx, "i", id)
	if err != nil || !again.CompletedAt.Equal(*done.CompletedAt) {
		t.Fatal(again, err)
	}
	// Tied timestamps must be ordered by ID by SQL, before state/next-task preparation.
	if _, err = db.Exec(`INSERT INTO tasks(id,intern_id,team,title,description,deadline,idempotency_key,created_at) VALUES ('b','i','backend','B','','2026-11-16','b','2026-11-15'),('a','i','backend','A','','2026-11-16','a','2026-11-15')`); err != nil {
		t.Fatal(err)
	}
	list, err = repo.Tasks(ctx, "i")
	if err != nil {
		t.Fatal(err)
	}
	var tied []string
	for _, task := range list {
		if task.ID == "a" || task.ID == "b" {
			tied = append(tied, task.ID)
		}
	}
	if len(tied) != 2 || tied[0] != "a" || tied[1] != "b" {
		t.Fatal(tied)
	}
	p := progress.New(repo, repo, loc, time.Now)
	if _, err = p.MentorProgress(ctx, "m", "j"); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatal(err)
	}
	if mentor, err := repo.Mentor(ctx, "j"); err != nil || mentor != nil {
		t.Fatal(mentor, err)
	}
	if empty, err := service.Tasks(ctx, "j"); err != nil || empty == nil || len(empty) != 0 {
		t.Fatal(empty, err)
	}
}
