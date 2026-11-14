package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/buddy/mcp"
	"ai-buddy/internal/buddy/ollama"
	"ai-buddy/internal/buddy/tracker"
	"ai-buddy/internal/platform/auth"
)

type repo struct {
	buddy.Repository
	mu       sync.Mutex
	events   []string
	text     string
	err      error
	failKind string
}

func (r *repo) History(context.Context, string, string) (buddy.History, error) {
	return buddy.History{Messages: []buddy.Message{{ID: "user", Role: "user", Content: "show progress"}, {ID: "old", Role: "user", Content: "old question"}, {RunID: "retry", Role: "assistant", Content: "old answer"}}, Runs: []buddy.Run{{ID: "retry", MessageID: "old", Status: "done"}}}, r.err
}
func (r *repo) Event(_ context.Context, _ string, kind string, v any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if kind == r.failKind || (kind == "stage" && v == r.failKind) {
		return errors.New("DB event failed")
	}
	r.events = append(r.events, kind)
	if kind == "text" {
		r.text += v.(string)
	}
	return nil
}

type progress struct{ err error }

func (p progress) Progress(context.Context, string) (tracker.Progress, error) {
	return tracker.Progress{Tasks: []tracker.Task{}}, p.err
}
func (p progress) Create(context.Context, string, string, json.RawMessage) (json.RawMessage, error) {
	return nil, nil
}
func (p progress) Complete(context.Context, string, string) (json.RawMessage, error) { return nil, nil }

type model struct {
	t       *testing.T
	step    int
	name    string
	args    string
	many    bool
	err     error
	block   chan struct{}
	entered chan struct{}
}

func (m *model) Chat(ctx context.Context, messages []ollama.Message, tools []ollama.Tool, stream bool, emit func(string) error) (ollama.Response, error) {
	m.step++
	if m.err != nil {
		return ollama.Response{}, m.err
	}
	if m.block != nil {
		select {
		case m.entered <- struct{}{}:
		default:
		}
		select {
		case <-m.block:
		case <-ctx.Done():
			return ollama.Response{}, ctx.Err()
		}
	}
	if stream {
		if len(tools) != 0 {
			m.t.Error("final has tools")
		}
		if e := emit("answer"); e != nil {
			return ollama.Response{}, e
		}
		return ollama.Response{Done: true, PromptTokens: 321}, nil
	}
	if len(tools) != 6 {
		m.t.Errorf("schemas not from MCP: %d", len(tools))
	}
	if m.step == 1 {
		if messages[1].Content != "old question" || messages[2].Content != "old answer" {
			m.t.Error(messages)
		}
	}
	if m.name != "" && (m.step == 1 || m.many) {
		args := m.args
		if args == "" {
			args = "{}"
		}
		calls := []ollama.Call{{Function: ollama.Function{Name: m.name, Arguments: json.RawMessage(args)}}}
		if m.many {
			for len(calls) < 7 {
				calls = append(calls, calls[0])
			}
		}
		return ollama.Response{Message: ollama.Message{Calls: calls}}, nil
	}
	return ollama.Response{Message: ollama.Message{Content: "ready"}}, nil
}
func TestGenerationAndFailures(t *testing.T) {
	for _, v := range []struct {
		name, args, err, kind string
		many                  bool
	}{{name: "get_progress"}, {name: "unknown", err: "invalid tool call"}, {name: "get_progress", args: "[]", err: "invalid tool arguments"}, {name: "get_progress", args: "not json", err: "invalid tool call"}, {name: "get_progress", args: `{"user_id":"bob"}`, err: "tool failed"}, {name: "get_progress", many: true, err: "tool call limit"}, {kind: "text", err: "DB event"}, {name: "get_progress", kind: "get_progress", err: "DB event"}, {kind: "answering", err: "DB event"}, {kind: "choosing_tools", err: "DB event"}, {kind: "waiting_for_model", err: "DB event"}} {
		t.Run(v.name+v.kind+v.args, func(t *testing.T) {
			r := &repo{failKind: v.kind}
			p := progress{}
			m := &model{t: t, name: v.name, args: v.args, many: v.many}
			a := New(r, &mcp.Tools{Repo: r, Tracker: p}, m)
			err := a.Generate(context.Background(), auth.User{ID: "alice"}, buddy.Run{ID: "r", ChatID: "c", MessageID: "user"})
			if v.err != "" {
				if err == nil || !strings.Contains(err.Error(), v.err) {
					t.Fatalf("want %s got %v", v.err, err)
				}
			} else if err != nil || r.text != "answer" {
				t.Fatal(err, r.text)
			}
		})
	}
	r := &repo{err: errors.New("DB history")}
	a := New(r, &mcp.Tools{Repo: r}, &model{t: t})
	if e := a.Generate(context.Background(), auth.User{}, buddy.Run{}); e == nil {
		t.Fatal("history")
	}
	r.err = nil
	a.Model = &model{t: t, err: errors.New("model unavailable")}
	if e := a.Generate(context.Background(), auth.User{}, buddy.Run{MessageID: "user"}); e == nil {
		t.Fatal("model")
	}
}
func TestBudgetPreservesCurrentRequest(t *testing.T) {
	messages := []ollama.Message{{Role: "system", Content: "system"}, {Role: "user", Content: strings.Repeat("old", 2000)}, {Role: "assistant", Content: "answer"}, {Role: "user", Content: "current"}, {Role: "assistant", Calls: []ollama.Call{{Function: ollama.Function{Name: "search", Arguments: json.RawMessage(`{}`)}}}}, {Role: "tool", Content: strings.Repeat("данные", 2000)}}
	out, e := Budget(messages, nil)
	if e != nil {
		t.Fatal(e)
	}
	if out[0].Content != "system" || out[1].Content != "current" {
		t.Fatal(out)
	}
	if _, e = Budget([]ollama.Message{{Role: "system", Content: strings.Repeat("a", 5000)}, {Role: "user", Content: "keep"}}, nil); e == nil {
		t.Fatal("overflow")
	}
	if clip("яблоко", 3) != "я…" {
		t.Fatal(clip("яблоко", 3))
	}
}
func TestSingleModelSlot(t *testing.T) {
	r := &repo{}
	block := make(chan struct{})
	entered := make(chan struct{}, 1)
	a := New(r, &mcp.Tools{Repo: r}, &model{t: t, block: block, entered: entered})
	done := make(chan error, 1)
	go func() { done <- a.Generate(context.Background(), auth.User{}, buddy.Run{MessageID: "user"}) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("not entered")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if e := a.Generate(ctx, auth.User{}, buddy.Run{MessageID: "user"}); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal(e)
	}
	close(block)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
}

// A model that keeps requesting tools must still get a final streaming answer
// after the sixth call, even when older tool rounds no longer fit the budget.
type repeatingModel struct{ selections int }

func (m *repeatingModel) Chat(_ context.Context, _ []ollama.Message, _ []ollama.Tool, stream bool, emit func(string) error) (ollama.Response, error) {
	if stream {
		return ollama.Response{Done: true}, emit("final")
	}
	m.selections++
	return ollama.Response{Message: ollama.Message{Calls: []ollama.Call{{Function: ollama.Function{Name: "get_progress", Arguments: json.RawMessage(`{}`)}}}}}, nil
}
func TestSixToolCallsStillProduceFinalAnswer(t *testing.T) {
	r := &repo{}
	model := &repeatingModel{}
	a := New(r, &mcp.Tools{Repo: r, Tracker: progress{}}, model)
	if e := a.Generate(context.Background(), auth.User{ID: "alice"}, buddy.Run{ID: "run", MessageID: "user"}); e != nil {
		t.Fatal(e)
	}
	if model.selections != 6 || r.text != "final" {
		t.Fatal(model.selections, r.text)
	}
}
