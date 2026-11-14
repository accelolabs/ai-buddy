package buddy

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"ai-buddy/internal/platform/auth"
)

type repoStub struct {
	Repository
	fresh     bool
	err       error
	fields    json.RawMessage
	execute   bool
	resultErr error
	finished  chan string
}

func (r *repoStub) Start(context.Context, string, string, string, string) (Run, bool, error) {
	return Run{ID: "run"}, r.fresh, r.err
}
func (r *repoStub) Retry(context.Context, string, string) (Run, bool, error) {
	return Run{ID: "run"}, r.fresh, r.err
}
func (r *repoStub) Finish(_ context.Context, _ string, e string) error { r.finished <- e; return nil }
func (r *repoStub) Claim(context.Context, string, string, json.RawMessage, bool) (Action, bool, error) {
	return Action{ID: "a", Status: "executing"}, r.execute, r.err
}
func (r *repoStub) ActionResult(_ context.Context, _ string, _ json.RawMessage, e error) (Action, error) {
	r.resultErr = e
	return Action{ID: "a"}, nil
}

type runner struct{ err error }

func (r runner) Generate(context.Context, auth.User, Run) error { return r.err }

type executor struct{ err error }

func (e executor) Execute(context.Context, auth.User, Action) (json.RawMessage, error) {
	return nil, e.err
}
func TestService(t *testing.T) {
	ctx := context.Background()
	u := auth.User{ID: "u"}
	r := &repoStub{fresh: true, execute: true, finished: make(chan string, 4)}
	s := &Service{Repo: r, Context: ctx, Runner: runner{}, Executor: executor{}}
	for _, v := range []struct{ k, text string }{{"", "hi"}, {strings.Repeat("k", 201), "hi"}, {"k", " "}, {"k", strings.Repeat("a", 2001)}} {
		if _, e := s.Send(ctx, u, "c", v.k, v.text); !errors.Is(e, ErrInvalid) {
			t.Fatal(e)
		}
	}
	if _, e := s.Send(ctx, u, "c", "k", "hi"); e != nil {
		t.Fatal(e)
	}
	s.Wait()
	if e := <-r.finished; e != "" {
		t.Fatal(e)
	}
	s.Runner = runner{errors.New("failed")}
	if _, e := s.Retry(ctx, u, "r"); e != nil {
		t.Fatal(e)
	}
	s.Wait()
	if e := <-r.finished; e != "failed" {
		t.Fatal(e)
	}
	r.fresh = false
	_, _ = s.Send(ctx, u, "c", "k", "hi")
	_, _ = s.Retry(ctx, u, "r")
	s.Wait()
	if _, e := s.Confirm(ctx, u, "a", nil, false); e != nil {
		t.Fatal(e)
	}
	s.Executor = executor{errors.New("network")}
	_, _ = s.Confirm(ctx, u, "a", nil, false)
	if r.resultErr == nil {
		t.Fatal("missing failure")
	}
	r.execute = false
	_, _ = s.Confirm(ctx, u, "a", nil, true)
	r.err = ErrNotFound
	if _, e := s.Confirm(ctx, u, "a", nil, false); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}
}
func TestFieldsAndModels(t *testing.T) {
	if len(ID()) != 32 || ID() == ID() {
		t.Fatal("ID")
	}
	if Title(" hello\n world ") != "hello world" || len([]rune(Title(strings.Repeat("я", 90)))) != 80 {
		t.Fatal("title")
	}
	for _, status := range []string{"queued", "running", "done", "error"} {
		if (Run{Status: status}).Active() != (status == "queued" || status == "running") {
			t.Fatal(status)
		}
	}
	for _, v := range []struct {
		kind, raw string
		valid     bool
	}{{"create_task", `{"title":"a","deadline":"2026-12-01"}`, true}, {"create_ticket", `{"question":"q","description":"d"}`, true}, {"other", `{}`, false}, {"create_task", `{"title":"","deadline":"2026-12-01"}`, false}, {"create_task", `{"title":"a","deadline":"tomorrow"}`, false}, {"create_task", `{"title":"a","deadline":"2026-12-01","user_id":"bob"}`, false}, {"create_ticket", `{"question":"q"}`, false}, {"create_ticket", `{} {}`, false}, {"create_task", `null`, false}} {
		if e := ValidateFields(v.kind, json.RawMessage(v.raw)); (e == nil) != v.valid {
			t.Fatalf("%s: %v", v.raw, e)
		}
	}
}
func TestLoad(t *testing.T) {
	for k, v := range map[string]string{"DATABASE_URL": "postgres://u:p@db/db", "OIDC_ISSUER": "https://auth.test/o/buddy/", "OIDC_CLIENT_ID": "buddy", "OIDC_CLIENT_SECRET": "secret", "OIDC_REDIRECT_URI": "https://buddy.test/api/auth/callback", "BUDDY_API_KEY": "key", "WIKI_PUBLIC_URL": "https://wiki.test"} {
		t.Setenv(k, v)
	}
	if _, e := Load(); e != nil {
		t.Fatal(e)
	}
	for k, v := range map[string]string{"OIDC_CLIENT_ID": "ticket", "BUDDY_API_KEY": " ", "WIKI_URL": "file:///x", "HTTP_ADDR": "bad"} {
		old := map[string]string{"OIDC_CLIENT_ID": "buddy", "BUDDY_API_KEY": "key", "WIKI_URL": "http://wiki:8080", "HTTP_ADDR": ":8080"}[k]
		t.Setenv(k, v)
		if _, e := Load(); e == nil {
			t.Fatal(k)
		}
		t.Setenv(k, old)
	}
	t.Setenv("SESSION_LIFETIME", "bad")
	if _, e := Load(); e == nil {
		t.Fatal("malformed")
	}
}
