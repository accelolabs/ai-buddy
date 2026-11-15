package buddy_test

import (
	"ai-buddy/internal/buddy"
	"ai-buddy/internal/ollama"
	"ai-buddy/internal/testutil"
	"ai-buddy/internal/ticket"
	"ai-buddy/internal/tracker"
	"ai-buddy/internal/users"
	"ai-buddy/internal/wiki"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type embedder struct{}

func (embedder) Fingerprint(context.Context) (string, error) { return "fixture", nil }
func (embedder) Embed(context.Context, string) ([]float64, error) {
	v := make([]float64, 768)
	v[0] = 1

	return v, nil
}

// A wrong vector dimension fails inside publication, exercising transaction rollback.
type invalidEmbedder struct{ embedder }

func (invalidEmbedder) Embed(
	context.Context,
	string,
) ([]float64, error) {
	return []float64{1}, nil
}

type scripted struct {
	tool, args string
	called     bool
}

func (m *scripted) Chat(
	_ context.Context,
	_ []ollama.Message,
	_ []ollama.Tool,
	emit func(string) error,
) (ollama.Response, error) {
	stream := emit != nil
	if stream {
		response := ollama.Response{Done: true, PromptTokens: 400}
		return response, emit("Verified response from tools.")
	}

	if m.called {
		return ollama.Response{}, nil
	}

	m.called = true

	return ollama.Response{
		Message: ollama.Message{
			Calls: []ollama.Call{
				{
					Function: ollama.Function{
						Name:      m.tool,
						Arguments: json.RawMessage(m.args),
					},
				},
			},
		},
	}, nil
}

// Simulate a tool result lost after a successful local commit.
type loseResponse struct {
	*tracker.Service
	lost bool
}

func (l *loseResponse) CreateTask(
	ctx context.Context,
	id, key string,
	in tracker.CreateInput,
) (tracker.Task, bool, error) {
	result, fresh, err := l.Service.CreateTask(ctx, id, key, in)
	if err == nil && !l.lost {
		l.lost = true
		return tracker.Task{}, false, errors.New("simulated lost result after commit")
	}

	return result, fresh, err
}
func TestServiceFlow(t *testing.T) { testFlow(t) }
func testFlow(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	db := testutil.Database(t)
	testutil.Indexing(t, db)

	_, e := db.ExecContext(
		context.Background(),
		`INSERT INTO profiles(id,name,username,email,role,team) VALUES ('alice','Alice','alice','a@test','intern','backend'),('bob','Bob','bob','b@test','intern','platform'),('mentor','Mentor','mentor','m@test','mentor','backend'); INSERT INTO assignments(intern_id,mentor_id,team) VALUES('alice','mentor','backend')`,
	)
	if e != nil {
		t.Fatal(e)
	}

	ir := &tracker.Store{DB: db}
	tr := ir
	ts := &tracker.Service{
		Store:    tr,
		Profiles: &users.Store{DB: db},
		Location: time.UTC,
		Now:      time.Now,
	}
	ticketService := ticket.New(&ticket.Store{DB: db}, &users.Store{DB: db})
	wr := &wiki.Store{DB: db}

	root := t.TempDir()
	if e = os.MkdirAll(filepath.Join(root, "common"), 0o755); e != nil {
		t.Fatal(e)
	}

	for team, text := range map[string]string{"backend": "Backend access process: ask the platform team for repository access.", "platform": "Platform private deployment secret."} {
		dir := filepath.Join(root, team)
		if e = os.MkdirAll(dir, 0o755); e != nil {
			t.Fatal(e)
		}

		if e = os.WriteFile(
			filepath.Join(dir, "process.md"),
			[]byte("# Access process\n\n"+text),
			0o644,
		); e != nil {
			t.Fatal(e)
		}
	}

	var embeddings wiki.Embedder = embedder{}
	if _, e = wiki.Run(ctx, wr, root, embeddings, false); e != nil {
		t.Fatal(e)
	}

	before, err := wr.List(ctx, "backend")
	if err != nil {
		t.Fatal(err)
	}

	if _, err = wiki.Run(ctx, wr, root, invalidEmbedder{}, true); err == nil {
		t.Fatal("expected failed publication")
	}

	after, err := wr.List(ctx, "backend")
	if err != nil || len(before) != len(after) {
		t.Fatal("index changed after rollback", after, err)
	}

	lock, err := wr.Acquire(ctx)
	if err != nil {
		t.Fatal("failed indexing retained lock", err)
	}

	if concurrent, err := wr.Acquire(ctx); err == nil {
		concurrent.Close()
		t.Fatal("concurrent index lock acquired")
	}

	lock.Close()

	released, err := wr.Acquire(ctx)
	if err != nil {
		t.Fatal("lock not released", err)
	}

	released.Close()

	repo := &buddy.Store{DB: db}
	trackerClient := &loseResponse{Service: ts}
	tools := &buddy.Tools{
		Repo: repo,
		Wiki: &wiki.Service{
			Store:    wr,
			Embedder: embeddings,
			Minimum:  0.1,
			Profiles: &users.Store{DB: db},
		},
		Tracker: trackerClient,
		Ticket:  ticketService,
	}

	for _, id := range []string{"alice", "bob"} {
		sources, e := tools.Wiki.Search(ctx, id, "access process")
		if e != nil || len(sources) == 0 {
			t.Fatal(sources, e)
		}

		for _, s := range sources {
			if id == "alice" && strings.Contains(s.Text, "secret") ||
				id == "bob" && strings.Contains(s.Text, "Backend") {
				t.Fatal("cross-team leak", sources)
			}
		}
	}

	u := users.User{ID: "alice", Role: "intern", Team: "backend"}

	var model buddy.Model

	run := func(request, tool, args string) buddy.History {
		t.Helper()

		model = &scripted{tool: tool, args: args}
		s := &buddy.Service{
			Repo:     repo,
			Runner:   buddy.NewAgent(repo, tools, model, "Тестовый системный промпт."),
			Executor: tools,
			Context:  ctx,
		}

		c, e := repo.CreateChat(ctx, u.ID)
		if e != nil {
			t.Fatal(e)
		}

		started := time.Now()

		r, e := s.Send(ctx, u, c.ID, buddy.ID(), request)
		if e != nil {
			t.Fatal(e)
		}

		s.Wait()

		h, e := repo.History(ctx, u.ID, c.ID)
		if e != nil {
			t.Fatal(e)
		}

		if h.Runs[0].Status != "done" {
			t.Fatalf("%s: %+v", request, h.Runs)
		}

		events, _, e := repo.Events(ctx, u.ID, r.ID, 0)
		if e != nil {
			t.Fatal(e)
		}

		for _, event := range events {
			if event.Kind == "usage" {
				var usage map[string]int
				if err := json.Unmarshal(event.Data, &usage); err != nil {
					t.Error(err)
				}

				t.Logf(
					"%q: latency=%s prompt_tokens=%d",
					request,
					time.Since(started).Round(time.Millisecond),
					usage["prompt_eval_count"],
				)

				if usage["prompt_eval_count"] > 3584 {
					t.Fatal("context reserve exceeded", usage)
				}
			}
		}

		if len(h.Messages) != 2 || h.Messages[1].Content == "" {
			t.Fatal(h.Messages)
		}

		return h
	}

	h := run(
		"How do I get repository access? Search Wiki.",
		"search_documents",
		`{"query":"repository access process"}`,
	)
	if len(h.Cards) == 0 || h.Cards[0].Kind != "source" {
		t.Fatal("missing source card", h)
	}

	run("Show my progress, tasks and mentor.", "get_progress", `{}`)

	h = run(
		"Prepare a task: title Read docs, description Read the onboarding guide, deadline 2026-12-01.",
		"create_task",
		`{"title":"Read docs","description":"Read the onboarding guide","deadline":"2026-12-01"}`,
	)
	if len(h.Actions) != 1 {
		t.Fatalf("expected draft: %+v", h)
	}

	s := &buddy.Service{Repo: repo, Executor: tools}

	a, e := s.Confirm(ctx, u, h.Actions[0].ID, nil)
	if e != nil || a.Status != "error" {
		t.Fatal(a, e)
	}

	a, e = s.Confirm(ctx, u, a.ID, nil)
	if e != nil || a.Status != "done" {
		t.Fatal(a, e)
	}

	again, e := s.Confirm(ctx, u, a.ID, nil)
	if e != nil || string(again.Result) != string(a.Result) {
		t.Fatal(again, e)
	}

	snapshot, e := ts.Progress(ctx, u.ID)

	own := snapshot.Tasks
	if e != nil || len(own) != 1 {
		t.Fatal(own, e)
	}

	run("Complete task "+own[0].ID, "complete_task", fmt.Sprintf(`{"task_id":%q}`, own[0].ID))
	snapshot, e = ts.Progress(ctx, u.ID)

	own = snapshot.Tasks
	if e != nil || own[0].CompletedAt == nil {
		t.Fatal(own, e)
	}

	h = run(
		"Prepare a mentor ticket: question Need help, description Please explain repository access.",
		"create_ticket",
		`{"question":"Need help","description":"Please explain repository access"}`,
	)
	if len(h.Actions) != 1 {
		t.Fatal("missing ticket draft")
	}

	a, e = s.Confirm(ctx, u, h.Actions[0].ID, nil)
	if e != nil || a.Status != "done" {
		t.Fatal(a, e)
	}

	_, e = s.Confirm(ctx, u, a.ID, nil)
	if e != nil {
		t.Fatal(e)
	}

	sent, e := ticketService.Sent(ctx, u.ID)
	if e != nil || len(sent) != 1 || sent[0].RecipientID != "mentor" {
		t.Fatal(sent, e)
	}

	if _, err := db.ExecContext(
		context.Background(),
		`DELETE FROM assignments WHERE intern_id='alice'`,
	); err != nil {
		t.Fatal(err)
	}

	var input ticket.CreateInput
	if err := json.Unmarshal(a.Fields, &input); err != nil {
		t.Fatal(err)
	}

	replayed, _, err := tools.Ticket.Create(ctx, u.ID, "buddy-action:"+a.ID, input)
	if err != nil {
		t.Fatal("ticket replay requires current mentor", err)
	}

	if replayed.RecipientID != "mentor" {
		t.Fatal(replayed, err)
	}

	run("List my sent tickets.", "list_tickets", `{}`)
}
