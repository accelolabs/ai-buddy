package buddy_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-buddy/internal/buddy/remote"
	"ai-buddy/internal/buddy/ticket"
	"ai-buddy/internal/buddy/tracker"
	"ai-buddy/internal/buddy/wiki"
)

func TestServiceClientsAndTeamScopedSearch(t *testing.T) {
	mode := "ok"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer buddy" {
			t.Error(r.Header)
		}
		if mode == "error" {
			w.WriteHeader(503)
			return
		}
		switch r.URL.Path {
		case "/internal/users/alice/search", "/internal/users/bob/search":
			var input map[string]string
			json.NewDecoder(r.Body).Decode(&input)
			if len(input) != 1 || input["query"] != "process" {
				t.Error(input)
			}
			if mode == "empty" {
				fmt.Fprint(w, `[]`)
				return
			}
			if mode == "null" {
				fmt.Fprint(w, `null`)
				return
			}
			if mode == "badurl" {
				fmt.Fprint(w, `[{"url":"https://evil.test/doc"}]`)
				return
			}
			team := "backend"
			if r.URL.Path == "/internal/users/bob/search" {
				team = "platform"
			}
			fmt.Fprintf(w, `[{"document_id":"%s","document_title":"Process","text":"Team %s","url":"/api/documents/%s"}]`, team, team, team)
		case "/internal/interns/alice/progress":
			fmt.Fprint(w, `{"tasks":[{"id":"task","title":"Read","status":"pending"}],"mentor":null,"total":1}`)
		case "/internal/interns/alice/tasks", "/internal/interns/alice/tickets":
			if r.Method == "POST" && r.Header.Get("Idempotency-Key") != "action-key" {
				t.Error(r.Header)
			}
			fmt.Fprint(w, `{"id":"created"}`)
		case "/internal/interns/alice/tasks/task/complete":
			fmt.Fprint(w, `{"id":"task","status":"completed"}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	remoteClient := remote.New(server.URL, "buddy", time.Second)
	w := &wiki.Client{Client: remoteClient, PublicURL: "https://wiki.test"}
	tr := &tracker.Client{Client: remoteClient}
	ti := &ticket.Client{Client: remoteClient}
	ctx := context.Background()
	for _, id := range []string{"alice", "bob"} {
		sources, e := w.Search(ctx, id, "process")
		if e != nil || len(sources) != 1 {
			t.Fatal(sources, e)
		}
		expected := "backend"
		if id == "bob" {
			expected = "platform"
		}
		if sources[0].DocumentID != expected || sources[0].URL != "https://wiki.test/api/documents/"+expected {
			t.Fatal(sources)
		}
	}
	p, e := tr.Progress(ctx, "alice")
	if e != nil || p.Total != 1 || len(p.Tasks) != 1 {
		t.Fatal(p, e)
	}
	fields := json.RawMessage(`{"title":"Read","deadline":"2026-12-01"}`)
	if _, e = tr.Create(ctx, "alice", "action-key", fields); e != nil {
		t.Fatal(e)
	}
	if _, e = tr.Complete(ctx, "alice", "task"); e != nil {
		t.Fatal(e)
	}
	if _, e = ti.Create(ctx, "alice", "action-key", fields); e != nil {
		t.Fatal(e)
	}
	if _, e = ti.List(ctx, "alice"); e != nil {
		t.Fatal(e)
	}
	for _, v := range []string{"empty", "null"} {
		mode = v
		out, e := w.Search(ctx, "alice", "process")
		if e != nil || out == nil || len(out) != 0 {
			t.Fatal(out, e)
		}
	}
	for _, v := range []string{"error", "badurl"} {
		mode = v
		if _, e = w.Search(ctx, "alice", "process"); e == nil {
			t.Fatal(v)
		}
	}
}
