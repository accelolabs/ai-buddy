package buddy_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/buddy/agent"
	"ai-buddy/internal/buddy/mcp"
	bmigrations "ai-buddy/internal/buddy/migrations"
	"ai-buddy/internal/buddy/ollama"
	bpostgres "ai-buddy/internal/buddy/postgres"
	"ai-buddy/internal/buddy/remote"
	bticket "ai-buddy/internal/buddy/ticket"
	btracker "ai-buddy/internal/buddy/tracker"
	bwiki "ai-buddy/internal/buddy/wiki"
	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/platform/httpx"
	platformmigrations "ai-buddy/internal/platform/migrations"
	"ai-buddy/internal/testutil"
	ticketmigrations "ai-buddy/internal/ticket/migrations"
	"ai-buddy/internal/ticket/tickets"
	ticketshttp "ai-buddy/internal/ticket/tickets/http"
	ticketpostgres "ai-buddy/internal/ticket/tickets/postgres"
	tickettracker "ai-buddy/internal/ticket/tracker"
	trackerhttp "ai-buddy/internal/tracker/http"
	trackermigrations "ai-buddy/internal/tracker/migrations"
	internspostgres "ai-buddy/internal/tracker/postgres"
	"ai-buddy/internal/tracker/progress"
	"ai-buddy/internal/tracker/tasks"
	"ai-buddy/internal/wiki"
	wikihttp "ai-buddy/internal/wiki/http"
	"ai-buddy/internal/wiki/indexer"
	wikimigrations "ai-buddy/internal/wiki/migrations"
	wikipostgres "ai-buddy/internal/wiki/postgres"
)

type teams struct{}

func (teams) Team(_ context.Context, id string) (string, error) {
	if id == "bob" {
		return "platform", nil
	}
	return "backend", nil
}

type embedder struct{}

func (embedder) Fingerprint(context.Context) (string, error) { return "fixture", nil }
func (embedder) Embed(context.Context, string) ([]float64, error) {
	v := make([]float64, 768)
	v[0] = 1
	return v, nil
}

type scripted struct {
	tool, args string
	called     bool
}

func (m *scripted) Chat(_ context.Context, _ []ollama.Message, _ []ollama.Tool, stream bool, emit func(string) error) (ollama.Response, error) {
	if stream {
		return ollama.Response{Done: true, PromptTokens: 400}, emit("Verified response from tools.")
	}
	if m.called {
		return ollama.Response{}, nil
	}
	m.called = true
	return ollama.Response{Message: ollama.Message{Calls: []ollama.Call{{Function: ollama.Function{Name: m.tool, Arguments: json.RawMessage(m.args)}}}}}, nil
}

type loseResponse struct{ lost bool }

func (l *loseResponse) RoundTrip(r *http.Request) (*http.Response, error) {
	res, e := http.DefaultTransport.RoundTrip(r)
	if e == nil && r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/tasks") && !l.lost {
		l.lost = true
		if err := res.Body.Close(); err != nil {
			return nil, err
		}
		return nil, errors.New("simulated lost response after commit")
	}
	return res, e
}
func TestServiceFlow(t *testing.T) { testFlow(t) }
func testFlow(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	db := testutil.Database(t, trackermigrations.Files)
	_, e := db.Exec(`INSERT INTO profiles(authentik_id,name,username,email,role,team) VALUES ('alice','Alice','alice','a@test','intern','backend'),('mentor','Mentor','mentor','m@test','mentor','backend'); INSERT INTO assignments(intern_id,mentor_id,team) VALUES('alice','mentor','backend')`)
	if e != nil {
		t.Fatal(e)
	}
	ir := internspostgres.New(db)
	tr := ir
	ts := tasks.New(tr, ir, time.UTC, time.Now)
	trackerHandler := trackerhttp.New(ts, ir, progress.New(ir, tr, time.UTC, time.Now))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /internal/interns/{id}/progress", trackerHandler.Progress)
	mux.HandleFunc("GET /internal/interns/{id}/mentor", trackerHandler.Mentor)
	mux.HandleFunc("POST /internal/interns/{id}/tasks", trackerHandler.Create)
	mux.HandleFunc("POST /internal/interns/{id}/tasks/{taskID}/complete", trackerHandler.Complete)
	trackerServer := httptest.NewServer(httpx.KeyAuth(mux, "buddy", "", false))
	defer trackerServer.Close()
	ticketDB := testutil.Database(t, ticketmigrations.Files)
	ticketService := tickets.New(ticketpostgres.New(ticketDB), tickettracker.New(trackerServer.URL, "buddy", 10*time.Second))
	ticketHandler := ticketshttp.New(ticketService)
	ticketMux := http.NewServeMux()
	ticketMux.HandleFunc("GET /internal/interns/{id}/tickets", ticketHandler.Sent)
	ticketMux.HandleFunc("POST /internal/interns/{id}/tickets", ticketHandler.Create)
	ticketServer := httptest.NewServer(httpx.KeyAuth(ticketMux, "buddy", "", false))
	defer ticketServer.Close()
	wikiDB := testutil.Database(t, wikimigrations.Files)
	for _, command := range []string{"status", "down", "up"} {
		if err := platformmigrations.Run(ctx, wikiDB, wikimigrations.Files, command); err != nil {
			t.Fatal(err)
		}
	}
	wr := wikipostgres.New(wikiDB)
	root := t.TempDir()
	if e = os.MkdirAll(filepath.Join(root, "common"), 0755); e != nil {
		t.Fatal(e)
	}
	for team, text := range map[string]string{"backend": "Backend access process: ask the platform team for repository access.", "platform": "Platform private deployment secret."} {
		dir := filepath.Join(root, team)
		if e = os.MkdirAll(dir, 0755); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(dir, "process.md"), []byte("# Access process\n\n"+text), 0644); e != nil {
			t.Fatal(e)
		}
	}
	var embeddings wiki.Embedder = embedder{}
	if _, e = indexer.Run(ctx, wr, root, embeddings, false); e != nil {
		t.Fatal(e)
	}
	wikiHandler := &wikihttp.Handler{Service: &wiki.Service{Store: wr, Embedder: embeddings, Minimum: 0.1}, Teams: teams{}}
	wikiMux := http.NewServeMux()
	wikiMux.HandleFunc("POST /internal/users/{id}/search", wikiHandler.Search)
	wikiServer := httptest.NewServer(httpx.KeyAuth(wikiMux, "buddy", "", false))
	defer wikiServer.Close()
	repo := bpostgres.New(testutil.Database(t, bmigrations.Files))
	trackerClient := remote.New(trackerServer.URL, "buddy", 10*time.Second)
	tools := &mcp.Tools{Repo: repo, Wiki: &bwiki.Client{Client: remote.New(wikiServer.URL, "buddy", 90*time.Second), PublicURL: "https://wiki.test"}, Tracker: &btracker.Client{Client: trackerClient}, Ticket: &bticket.Client{Client: remote.New(ticketServer.URL, "buddy", 10*time.Second)}}
	for _, id := range []string{"alice", "bob"} {
		sources, e := tools.Wiki.Search(ctx, id, "access process")
		if e != nil || len(sources) == 0 {
			t.Fatal(sources, e)
		}
		for _, s := range sources {
			if id == "alice" && strings.Contains(s.Text, "secret") || id == "bob" && strings.Contains(s.Text, "Backend") {
				t.Fatal("cross-team leak", sources)
			}
		}
	}
	u := auth.User{ID: "alice", Role: "intern", Team: "backend"}
	var model agent.Model
	run := func(request, tool, args string) buddy.History {
		t.Helper()
		model = &scripted{tool: tool, args: args}
		s := &buddy.Service{Repo: repo, Runner: agent.New(repo, tools, model), Executor: tools, Context: ctx}
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
				t.Logf("%q: latency=%s prompt_tokens=%d", request, time.Since(started).Round(time.Millisecond), usage["prompt_eval_count"])
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
	h := run("How do I get repository access? Search Wiki.", "search_documents", `{"query":"repository access process"}`)
	if len(h.Cards) == 0 || h.Cards[0].Kind != "source" {
		t.Fatal("missing source card", h)
	}
	run("Show my progress, tasks and mentor.", "get_progress", `{}`)
	h = run("Prepare a task: title Read docs, description Read the onboarding guide, deadline 2026-12-01.", "create_task", `{"title":"Read docs","description":"Read the onboarding guide","deadline":"2026-12-01"}`)
	if len(h.Actions) != 1 {
		t.Fatalf("expected draft: %+v", h)
	}
	loss := &loseResponse{}
	trackerClient.HTTP.Transport = loss
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
	own, e := ts.Tasks(ctx, u.ID)
	if e != nil || len(own) != 1 {
		t.Fatal(own, e)
	}
	run("Complete task "+own[0].ID, "complete_task", fmt.Sprintf(`{"task_id":%q}`, own[0].ID))
	own, e = ts.Tasks(ctx, u.ID)
	if e != nil || own[0].CompletedAt == nil {
		t.Fatal(own, e)
	}
	h = run("Prepare a mentor ticket: question Need help, description Please explain repository access.", "create_ticket", `{"question":"Need help","description":"Please explain repository access"}`)
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
	run("List my sent tickets.", "list_tickets", `{}`)
}
