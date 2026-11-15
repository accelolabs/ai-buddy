package buddy

import (
	"ai-buddy/internal/migrations"
	"ai-buddy/internal/testutil"
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func repository(t *testing.T) *Store {
	t.Helper()
	p := &Store{DB: testutil.Database(t)}
	storeFixture(t, p)

	return p
}

func must(t *testing.T, e error) {
	t.Helper()

	if e != nil {
		t.Fatal(e)
	}
}

func storeFixture(t *testing.T, p *Store) {
	t.Helper()

	ctx := context.Background()

	_, err := p.DB.ExecContext(
		ctx,
		`INSERT INTO profiles(id,name,username,email,role,team) VALUES ('alice','Alice','alice','a@test','intern','backend'),('bob','Bob','bob','b@test','intern','platform')`,
	)
	must(t, err)
}

var task = json.RawMessage(`{"title":"Read docs","description":"Intro","deadline":"2026-12-01"}`)

func TestLifecycleIsolationReplayAndRestart(t *testing.T) {
	ctx := context.Background()
	p := repository(t)
	c, e := p.CreateChat(ctx, "alice")
	must(t, e)
	chats, e := p.Chats(ctx, "bob")
	must(t, e)

	if len(chats) != 0 {
		t.Fatal(chats)
	}

	if _, e = p.History(ctx, "bob", c.ID); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}

	r, fresh, e := p.Start(ctx, "alice", c.ID, "key", " First\n message ")
	must(t, e)

	if !fresh {
		t.Fatal("not fresh")
	}

	same, fresh, e := p.Start(ctx, "alice", c.ID, "key", " First\n message ")
	must(t, e)

	if fresh || same.ID != r.ID {
		t.Fatal(same)
	}

	if _, _, e = p.Start(ctx, "alice", c.ID, "key", "changed"); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}

	if _, _, e = p.Start(ctx, "alice", c.ID, "other", "new"); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}

	if _, _, e = p.Start(ctx, "bob", c.ID, "key", "x"); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}

	if _, _, e = p.Retry(ctx, "alice", r.ID); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}

	if _, e = p.Run(ctx, "bob", r.ID); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}

	must(t, p.Event(ctx, r.ID, "stage", "answering"))
	must(t, p.Event(ctx, r.ID, "text", "hello "))
	must(t, p.Event(ctx, r.ID, "text", "world"))

	if e = p.Event(ctx, r.ID, "text", 42); !errors.Is(e, ErrInvalid) {
		t.Fatal(e)
	}

	if e = p.Event(ctx, r.ID, "bad", make(chan int)); e == nil {
		t.Fatal("marshal should fail")
	}

	a, e := p.Draft(ctx, Action{RunID: r.ID, Owner: "alice", Kind: "create_task", Fields: task})
	must(t, e)

	if _, e = p.Draft(
		ctx,
		Action{RunID: r.ID, Owner: "bob", Kind: "create_task", Fields: task},
	); !errors.Is(
		e,
		ErrNotFound,
	) {
		t.Fatal(e)
	}

	h, e := p.History(ctx, "alice", c.ID)
	must(t, e)

	if h.Chat.Title != "First message" || len(h.Messages) != 2 ||
		h.Messages[1].Content != "hello world" ||
		len(h.Actions) != 1 ||
		len(h.Runs) != 1 {
		t.Fatalf("%+v", h)
	}

	must(t, p.Recover(ctx))
	events, got, e := p.Events(ctx, "alice", r.ID, 1)
	must(t, e)

	if got.Status != "error" || len(events) != 3 || events[2].Kind != "error" {
		t.Fatalf("%+v %+v", events, got)
	}

	if _, _, e = p.Events(ctx, "bob", r.ID, 0); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}

	if e = p.Event(ctx, r.ID, "text", "later"); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}

	must(t, p.Finish(ctx, r.ID, ""))
	next, fresh, e := p.Retry(ctx, "alice", r.ID)
	must(t, e)

	if !fresh || next.MessageID != r.MessageID {
		t.Fatal(next)
	}

	repeated, fresh, e := p.Retry(ctx, "alice", r.ID)
	must(t, e)

	if fresh || repeated.ID != next.ID {
		t.Fatal(repeated)
	}

	must(t, p.Finish(ctx, next.ID, ""))
	h, e = p.History(ctx, "alice", c.ID)
	must(t, e)

	if len(h.Messages) != 3 {
		t.Fatal(h.Messages)
	}

	chats, e = p.Chats(ctx, "alice")
	must(t, e)

	if len(chats) != 1 {
		t.Fatal(chats)
	}

	if _, _, e = p.Claim(ctx, "bob", a.ID, nil); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}

	if e = p.Finish(ctx, "absent", ""); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}

	if e = p.Event(ctx, "absent", "stage", "x"); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}

	must(t, migrations.Run(ctx, p.DB, "down"))
	must(t, migrations.Run(ctx, p.DB, "up"))
	storeFixture(t, p)
}

func TestActionsFreezeCancelRetryAndRecovery(t *testing.T) {
	ctx := context.Background()
	p := repository(t)
	c, e := p.CreateChat(ctx, "alice")
	must(t, e)
	r, _, e := p.Start(ctx, "alice", c.ID, "k", "hello")
	must(t, e)

	draft := func() Action {
		a, e := p.Draft(
			ctx,
			Action{Owner: "alice", RunID: r.ID, Kind: "create_task", Fields: task},
		)
		must(t, e)

		return a
	}

	a := draft()
	if _, _, e = p.Claim(
		ctx,
		"alice",
		a.ID,
		json.RawMessage(`{"title":""}`),
	); !errors.Is(
		e,
		ErrInvalid,
	) {
		t.Fatal(e)
	}

	edited := json.RawMessage(`{"title":"Edited","description":"d","deadline":"2026-12-02"}`)
	claimed, execute, e := p.Claim(ctx, "alice", a.ID, edited)
	must(t, e)

	if !execute || claimed.Status != "executing" {
		t.Fatal(claimed)
	}

	if _, _, e = p.Claim(ctx, "alice", a.ID, nil); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}

	if _, e = p.Cancel(ctx, "alice", a.ID); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}

	failed, e := p.ActionResult(ctx, a.ID, nil, errors.New("lost response"))
	must(t, e)

	if failed.Status != "error" {
		t.Fatal(failed)
	}

	if _, _, e = p.Claim(ctx, "alice", a.ID, task); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}

	claimed, execute, e = p.Claim(ctx, "alice", a.ID, edited)
	must(t, e)

	if !execute || claimed.ID != a.ID {
		t.Fatal(claimed)
	}

	done, e := p.ActionResult(ctx, a.ID, json.RawMessage(`{"id":"created"}`), nil)
	must(t, e)

	if done.Status != "done" {
		t.Fatal(done)
	}

	done, execute, e = p.Claim(ctx, "alice", a.ID, nil)
	must(t, e)

	if execute || done.Status != "done" {
		t.Fatal(done)
	}

	a = draft()
	cancelled, e := p.Cancel(ctx, "alice", a.ID)
	must(t, e)

	if cancelled.Status != "cancelled" {
		t.Fatal(cancelled)
	}

	_, e = p.Cancel(ctx, "alice", a.ID)
	must(t, e)

	if _, _, e = p.Claim(ctx, "alice", a.ID, nil); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}

	a = draft()
	_, _, e = p.Claim(ctx, "alice", a.ID, nil)
	must(t, e)
	must(t, p.Recover(ctx))
	claimed, execute, e = p.Claim(ctx, "alice", a.ID, nil)
	must(t, e)

	if !execute || claimed.ID != a.ID {
		t.Fatal(claimed)
	}
}
