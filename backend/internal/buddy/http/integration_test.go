package buddyhttp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/buddy/migrations"
	"ai-buddy/internal/buddy/postgres"
	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/testutil"

	"github.com/alexedwards/scs/v2"
)

type runner struct{ repo buddy.Repository }

func (r runner) Generate(ctx context.Context, _ auth.User, run buddy.Run) error {
	if e := r.repo.Event(ctx, run.ID, "stage", "answering"); e != nil {
		return e
	}
	return r.repo.Event(ctx, run.ID, "text", "answer")
}

type executor struct{}

func (executor) Execute(_ context.Context, _ auth.User, a buddy.Action) (json.RawMessage, error) {
	if a.Kind == "create_ticket" {
		return json.RawMessage(`{"id":"created","sender_id":"alice","recipient_id":"mentor","question":"Help?","description":"context","created_at":"2026-11-14T12:00:00Z"}`), nil
	}
	return json.RawMessage(`{"id":"created","intern_id":"alice","title":"Read","description":"","deadline":"2026-12-01","created_at":"2026-11-14T12:00:00Z","completed_at":null,"status":"pending","overdue":false}`), nil
}
func TestHTTPWithSessionsAndPersistedSSE(t *testing.T) {
	ctx := context.Background()
	p := postgres.New(testutil.Database(t, migrations.Files))
	s := &buddy.Service{Repo: p, Runner: runner{p}, Executor: executor{}, Context: ctx}
	defer s.Wait()
	sessions := scs.New()
	login := auth.New(sessions, nil)
	handler := New(s)
	handler.Poll = time.Millisecond
	handler.Heartbeat = time.Millisecond
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := sessions.Load(r.Context(), "")
		u := auth.User{ID: r.URL.Query().Get("id"), Role: r.URL.Query().Get("role"), Team: "backend"}
		b, _ := json.Marshal(u)
		sessions.Put(ctx, auth.SessionKeyUser, string(b))
		token, _, e := sessions.Commit(ctx)
		if e != nil {
			t.Error(e)
		}
		http.SetCookie(w, &http.Cookie{Name: "session", Value: token})
	})
	mux.Handle("/", handler.Routes(login))
	server := httptest.NewServer(mux)
	defer server.Close()
	cookie := func(id, role string) *http.Cookie {
		res, e := http.Get(server.URL + "/login?id=" + id + "&role=" + role)
		if e != nil {
			t.Fatal(e)
		}
		defer func() {
			if err := res.Body.Close(); err != nil {
				t.Error(err)
			}
		}()
		return res.Cookies()[0]
	}
	alice, bob, mentor := cookie("alice", "intern"), cookie("bob", "intern"), cookie("mentor", "mentor")
	call := func(method, path, body string, c *http.Cookie, key string, code int) []byte {
		t.Helper()
		req, _ := http.NewRequest(method, server.URL+path, strings.NewReader(body))
		if c != nil {
			req.AddCookie(c)
		}
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		res, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer func() {
			if err := res.Body.Close(); err != nil {
				t.Error(err)
			}
		}()
		b, e := io.ReadAll(res.Body)
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != code {
			t.Fatalf("%s %s: %d %s", method, path, res.StatusCode, b)
		}
		return b
	}
	call("GET", "/api/chats", "", nil, "", 401)
	call("GET", "/api/chats", "", mentor, "", 403)
	var c chatDTO
	if e := json.Unmarshal(call("POST", "/api/chats", "", alice, "", 201), &c); e != nil {
		t.Fatal(e)
	}
	call("GET", "/api/chats", "", alice, "", 200)
	call("GET", "/api/chats/"+c.ID, "", bob, "", 404)
	call("POST", "/api/chats/"+c.ID+"/messages", `{"content":"hi","owner":"bob"}`, alice, "k", 400)
	call("POST", "/api/chats/"+c.ID+"/messages", `{"content":"hi"}`, alice, "", 400)
	var r runDTO
	if err := json.Unmarshal(call("POST", "/api/chats/"+c.ID+"/messages", `{"content":"hi"}`, alice, "k", 202), &r); err != nil {
		t.Error(err)
	}
	s.Wait()
	var same runDTO
	if err := json.Unmarshal(call("POST", "/api/chats/"+c.ID+"/messages", `{"content":"hi"}`, alice, "k", 202), &same); err != nil {
		t.Error(err)
	}
	if same.ID != r.ID {
		t.Fatal(same)
	}
	call("POST", "/api/chats/"+c.ID+"/messages", `{"content":"changed"}`, alice, "k", 409)
	call("GET", r.EventsURL, "", bob, "", 404)
	call("POST", "/api/runs/"+r.ID+"/retry", "", alice, "", 409)
	call("POST", "/api/runs/"+r.ID+"/retry", "", bob, "", 404)
	late := call("GET", r.EventsURL, "", alice, "", 200)
	if !strings.Contains(string(late), "event: text") || !strings.Contains(string(late), "event: done") {
		t.Fatal(string(late))
	}
	req, _ := http.NewRequest("GET", server.URL+r.EventsURL, nil)
	req.AddCookie(alice)
	req.Header.Set("Last-Event-ID", "2")
	res, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	replay, _ := io.ReadAll(res.Body)
	if err := res.Body.Close(); err != nil {
		t.Error(err)
	}
	if strings.Contains(string(replay), "event: text") || !strings.Contains(string(replay), "event: done") {
		t.Fatal(string(replay))
	}
	req.Header.Set("Last-Event-ID", "bad")
	res, e = http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	if err := res.Body.Close(); err != nil {
		t.Error(err)
	}
	if res.StatusCode != 400 {
		t.Fatal(res.StatusCode)
	}
	c2, e := p.CreateChat(ctx, "alice")
	if e != nil {
		t.Fatal(e)
	}
	live, _, e := p.Start(ctx, "alice", c2.ID, "live", "hello")
	if e != nil {
		t.Fatal(e)
	}
	req, _ = http.NewRequest("GET", server.URL+"/api/runs/"+live.ID+"/events", nil)
	req.AddCookie(alice)
	res, e = http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	reader := bufio.NewReader(res.Body)
	line, e := reader.ReadString('\n')
	if e != nil || !strings.HasPrefix(line, ": heartbeat") {
		t.Fatal(line, e)
	}
	if err := res.Body.Close(); err != nil {
		t.Error(err)
	}
	if e = p.Event(ctx, live.ID, "text", "persisted after disconnect"); e != nil {
		t.Fatal(e)
	}
	a, e := p.Draft(ctx, buddy.Action{RunID: live.ID, Owner: "alice", Kind: "create_task", Fields: json.RawMessage(`{"title":"Read","deadline":"2026-12-01"}`)})
	if e != nil {
		t.Fatal(e)
	}
	call("POST", "/api/actions/"+a.ID+"/confirm", `{}`, bob, "", 404)
	call("POST", "/api/actions/"+a.ID+"/confirm", `{"fields":{"title":"Edited","deadline":"2026-12-02"}}`, alice, "", 200)
	call("POST", "/api/actions/"+a.ID+"/confirm", `{}`, alice, "", 200)
	call("POST", "/api/actions/"+a.ID+"/cancel", "", alice, "", 409)
	call("POST", "/api/actions/"+a.ID+"/confirm", `{"user":"bob"}`, alice, "", 400)
	a, e = p.Draft(ctx, buddy.Action{RunID: live.ID, Owner: "alice", Kind: "create_ticket", Fields: json.RawMessage(`{"question":"Help?","description":"context"}`)})
	if e != nil {
		t.Fatal(e)
	}
	call("POST", "/api/actions/"+a.ID+"/cancel", "", alice, "", 200)
	if e = p.Event(ctx, live.ID, "source", map[string]string{"document_id": "doc", "document_title": "Guide", "section_title": "Intro", "text": "Excerpt", "url": "https://wiki.example.invalid/api/documents/doc"}); e != nil {
		t.Fatal(e)
	}
	if e = p.Finish(ctx, live.ID, "failed"); e != nil {
		t.Fatal(e)
	}
	history := call("GET", "/api/chats/"+c2.ID, "", alice, "", 200)
	for _, want := range []string{"persisted after disconnect", "cards", "Edited", "cancelled", "latest_run"} {
		if !strings.Contains(string(history), want) {
			t.Fatal(want, string(history))
		}
	}
	call("POST", "/api/runs/"+live.ID+"/retry", "", alice, "", 202)
	s.Wait()
	call("DELETE", "/api/chats", "", alice, "", 405)
	call("GET", "/api/unknown", "", alice, "", 404)
}
