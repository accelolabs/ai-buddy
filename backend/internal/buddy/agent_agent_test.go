package buddy

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"ai-buddy/internal/ollama"
	"ai-buddy/internal/tracker"
	"ai-buddy/internal/users"
)

type agentRepo struct {
	Repository
	mu       sync.Mutex
	events   []string
	text     string
	err      error
	failKind string
}

func (r *agentRepo) History(context.Context, string, string) (History, error) {
	return History{Messages: []Message{{ID: "user", Role: "user", Content: "show progress"}, {ID: "old", Role: "user", Content: "old question"}, {RunID: "retry", Role: "assistant", Content: "old answer"}}, Runs: []Run{{ID: "retry", MessageID: "old", Status: "done"}}}, r.err
}
func (r *agentRepo) Event(_ context.Context, _ string, kind string, v any) error {
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

type agentProgress struct{ err error }

func (p agentProgress) Progress(context.Context, string) (tracker.Progress, error) {
	return tracker.Progress{Tasks: []tracker.Task{}}, p.err
}
func (p agentProgress) CreateTask(context.Context, string, string, tracker.CreateInput) (tracker.Task, bool, error) {
	return tracker.Task{}, false, nil
}
func (p agentProgress) CompleteTask(context.Context, string, string) (tracker.Task, error) {
	return tracker.Task{}, nil
}

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

func (m *model) Chat(ctx context.Context, messages []ollama.Message, tools []ollama.Tool, emit func(string) error) (ollama.Response, error) {
	stream := emit != nil
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
			r := &agentRepo{failKind: v.kind}
			p := agentProgress{}
			m := &model{t: t, name: v.name, args: v.args, many: v.many}
			a := NewAgent(r, &Tools{Repo: r, Tracker: p}, m)
			err := a.Generate(context.Background(), users.User{ID: "alice"}, Run{ID: "r", ChatID: "c", MessageID: "user"})
			if v.err != "" {
				if err == nil || !strings.Contains(err.Error(), v.err) {
					t.Fatalf("want %s got %v", v.err, err)
				}
			} else if err != nil || r.text != "answer" {
				t.Fatal(err, r.text)
			}
		})
	}
	r := &agentRepo{err: errors.New("DB history")}
	a := NewAgent(r, &Tools{Repo: r}, &model{t: t})
	if e := a.Generate(context.Background(), users.User{}, Run{}); e == nil {
		t.Fatal("history")
	}
	r.err = nil
	a.Model = &model{t: t, err: errors.New("model unavailable")}
	if e := a.Generate(context.Background(), users.User{}, Run{MessageID: "user"}); e == nil {
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
	r := &agentRepo{}
	block := make(chan struct{})
	entered := make(chan struct{}, 1)
	a := NewAgent(r, &Tools{Repo: r}, &model{t: t, block: block, entered: entered})
	done := make(chan error, 1)
	go func() { done <- a.Generate(context.Background(), users.User{}, Run{MessageID: "user"}) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("not entered")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if e := a.Generate(ctx, users.User{}, Run{MessageID: "user"}); !errors.Is(e, context.DeadlineExceeded) {
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

func (m *repeatingModel) Chat(_ context.Context, _ []ollama.Message, _ []ollama.Tool, emit func(string) error) (ollama.Response, error) {
	stream := emit != nil
	if stream {
		return ollama.Response{Done: true}, emit("final")
	}
	m.selections++
	return ollama.Response{Message: ollama.Message{Calls: []ollama.Call{{Function: ollama.Function{Name: "get_progress", Arguments: json.RawMessage(`{}`)}}}}}, nil
}
func TestSixToolCallsStillProduceFinalAnswer(t *testing.T) {
	r := &agentRepo{}
	model := &repeatingModel{}
	a := NewAgent(r, &Tools{Repo: r, Tracker: agentProgress{}}, model)
	if e := a.Generate(context.Background(), users.User{ID: "alice"}, Run{ID: "run", MessageID: "user"}); e != nil {
		t.Fatal(e)
	}
	if model.selections != 6 || r.text != "final" {
		t.Fatal(model.selections, r.text)
	}
}
