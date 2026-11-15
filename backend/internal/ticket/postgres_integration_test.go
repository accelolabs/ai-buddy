package ticket

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"ai-buddy/internal/migrations"
	"ai-buddy/internal/testutil"
)

func TestPostgresIntegration(t *testing.T) {
	db := testutil.Database(t)
	for _, command := range []string{"up", "status", "down", "up"} {
		if err := migrations.Run(context.Background(), db, command); err != nil {
			t.Fatal(command, err)
		}
	}
	for _, id := range []string{"s", "m", "replacement", "other", "second"} {
		if _, err := db.Exec(`INSERT INTO profiles(id,name,username,email,role,team) VALUES ($1,$1,$1,$1,'intern','backend')`, id); err != nil {
			t.Fatal(err)
		}
	}
	repo := &Store{DB: db}
	ctx := context.Background()
	input := CreateInput{Question: "q", Description: "d"}
	var wg sync.WaitGroup
	var created atomic.Int32
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, fresh, err := repo.Create(ctx, "s", "m", "shared", input)
			if err != nil {
				t.Error(err)
			}
			if fresh {
				created.Add(1)
			}
		}()
	}
	wg.Wait()
	if created.Load() != 1 {
		t.Fatal(created.Load())
	}
	original, err := repo.Find(ctx, "s", "shared")
	if err != nil {
		t.Fatal(err)
	}
	repeated, fresh, err := repo.Create(ctx, "s", "replacement", "shared", input)
	if err != nil || fresh || repeated.ID != original.ID || repeated.RecipientID != "m" {
		t.Fatal(repeated, fresh, err)
	}
	_, _, err = repo.Create(ctx, "s", "m", "shared", CreateInput{Question: "changed", Description: "d"})
	if !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	for _, v := range []struct{ s, r, k string }{{"other", "m", "shared"}, {"s", "second", "second"}} {
		if _, _, err := repo.Create(ctx, v.s, v.r, v.k, input); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		fn    func(context.Context, string) ([]Ticket, error)
		id    string
		count int
	}{{repo.Sent, "s", 2}, {repo.Sent, "other", 1}, {repo.Incoming, "m", 2}, {repo.Incoming, "second", 1}, {repo.Incoming, "replacement", 0}, {repo.Sent, "none", 0}} {
		list, err := tc.fn(ctx, tc.id)
		if err != nil || list == nil || len(list) != tc.count {
			t.Fatal(list, err)
		}
		for i := 1; i < len(list); i++ {
			if list[i].CreatedAt.Before(list[i-1].CreatedAt) || (list[i].CreatedAt.Equal(list[i-1].CreatedAt) && list[i].ID < list[i-1].ID) {
				t.Fatal("unordered list")
			}
		}
	}
	// A tied timestamp exercises the ID tie-breaker.
	if _, err := db.ExecContext(ctx, "UPDATE tickets SET created_at='2026-01-01'"); err != nil {
		t.Fatal(err)
	}
	list, err := repo.Incoming(ctx, "m")
	if err != nil || list[0].ID > list[1].ID {
		t.Fatal(list, err)
	}
}
