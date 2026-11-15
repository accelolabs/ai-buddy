package buddy

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type repoStub struct {
	Repository
	fresh     bool
	err       error
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
func (r *repoStub) Claim(context.Context, string, string, json.RawMessage) (Action, bool, error) {
	return Action{ID: "a", Status: "executing"}, r.execute, r.err
}
func (r *repoStub) ActionResult(_ context.Context, _ string, _ json.RawMessage, e error) (Action, error) {
	r.resultErr = e
	return Action{ID: "a"}, nil
}

func TestFieldsAndModels(t *testing.T) {
	first, second := ID(), ID()
	if len(first) != 32 || first == second {
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
		if _, e := parseFields(v.kind, json.RawMessage(v.raw)); (e == nil) != v.valid {
			t.Fatalf("%s: %v", v.raw, e)
		}
	}
}
func (r *repoStub) Cancel(context.Context, string, string) (Action, error) { return Action{}, r.err }
