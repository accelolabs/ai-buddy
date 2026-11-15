package agent

import (
	"backend/internal/buddy"
	"backend/internal/ollama"
	"backend/internal/users"
	"backend/internal/wiki"
	"backend/internal/wiki/rag"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeModel struct {
	calls int
	reply func([]ollama.Message, []ollama.Tool) (ollama.Response, error)
}

func (m *fakeModel) Chat(_ context.Context, messages []ollama.Message, tools []ollama.Tool, emit func(string) error) (ollama.Response, error) {
	m.calls++
	response, err := m.reply(messages, tools)
	if emit == nil {
		panic("model call must stream")
	}
	if err == nil && response.Message.Content != "" {
		err = emit(response.Message.Content)
	}
	return response, err
}

type fakeRAG struct{ query, team string }

func (r *fakeRAG) Search(_ context.Context, query, team string) ([]rag.Match, error) {
	r.query, r.team = query, team
	return []rag.Match{{Document: wiki.Document{Title: "Title"}, Chunk: wiki.Chunk{DocumentID: "doc", Text: "Документ"}}}, nil
}

func testAgent(t *testing.T, model *fakeModel, rag *fakeRAG) *Agent {
	t.Helper()
	path := filepath.Join(t.TempDir(), "prompt.md")
	if err := os.WriteFile(path, []byte("Отвечай по-русски"), 0o600); err != nil {
		t.Fatal(err)
	}
	agent, err := NewAgent(model, rag, path)
	if err != nil {
		t.Fatal(err)
	}
	return agent
}

func TestGenerate(t *testing.T) {
	rag := &fakeRAG{}
	model := &fakeModel{}
	model.reply = func(messages []ollama.Message, tools []ollama.Tool) (ollama.Response, error) {
		if messages[0].Role != "system" || messages[0].Content != "Отвечай по-русски" {
			t.Fatal("missing prompt")
		}
		if model.calls == 1 {
			if len(tools) != 2 {
				t.Fatal("missing tools")
			}
			return ollama.Response{Message: ollama.Message{Calls: []ollama.Call{{Function: ollama.Function{Name: "search_documents", Arguments: []byte(`{"query":"question"}`)}}}}}, nil
		}
		last := messages[len(messages)-1]
		if last.Role != "tool" || last.ToolName != "search_documents" || !strings.Contains(last.Content, "Документ") {
			t.Fatalf("tool result: %+v", last)
		}
		return ollama.Response{Message: ollama.Message{Content: "Ответ"}}, nil
	}
	history := make([]buddy.Message, 1, 10)
	history[0] = buddy.Message{Role: "user", Content: "Вопрос"}
	var emitted string
	var states []string
	answer, err := testAgent(t, model, rag).Generate(t.Context(), users.User{ID: "user", Team: "backend"}, history, func(event buddy.Event) error {
		if event.Type == "state" {
			states = append(states, event.State+":"+event.Progress)
		}
		if event.Type == "text" {
			emitted += event.Text
		}
		return nil
	})
	if err != nil || answer.Content != "Ответ" || answer.Source != nil || model.calls != 2 || emitted != answer.Content || rag.team != "backend" || rag.query != "question" {
		t.Fatalf("answer: %+v, %v", answer, err)
	}
	if strings.Join(states, ",") != "thinking:,progress:search_documents,thinking:,answering:" {
		t.Fatalf("states: %v", states)
	}
	if history[:2][1].Role != "" {
		t.Fatal("history modified")
	}
}

func TestGenerateFailures(t *testing.T) {
	for _, name := range []string{"unknown", "arguments", "limit", "model", "progress"} {
		t.Run(name, func(t *testing.T) {
			rag := &fakeRAG{}
			failure := errors.New("model failed")
			model := &fakeModel{reply: func(_ []ollama.Message, _ []ollama.Tool) (ollama.Response, error) {
				call := ollama.Call{Function: ollama.Function{Name: "search_documents", Arguments: []byte(`{}`)}}
				switch name {
				case "unknown":
					call.Function.Name = "unknown"
				case "arguments":
					call.Function.Arguments = []byte(`null`)
				case "model":
					return ollama.Response{}, failure
				}
				calls := []ollama.Call{call}
				if name == "limit" {
					for range 6 {
						calls = append(calls, call)
					}
				}
				return ollama.Response{Message: ollama.Message{Calls: calls}}, nil
			}}
			var emit func(buddy.Event) error
			if name == "progress" {
				emit = func(event buddy.Event) error {
					if event.State == "progress" {
						return failure
					}
					return nil
				}
			}
			_, err := testAgent(t, model, rag).Generate(t.Context(), users.User{ID: "user", Team: "backend"}, nil, emit)
			if err == nil {
				t.Fatal("expected error")
			}
			if name == "model" && !errors.Is(err, failure) {
				t.Fatal(err)
			}
			if rag.query != "" {
				t.Fatal("unexpected tool execution")
			}
		})
	}
	if _, err := NewAgent(&fakeModel{}, &fakeRAG{}, filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing prompt accepted")
	}
}

func TestGenerateBindsCurrentUser(t *testing.T) {
	rag := &fakeRAG{}
	model := &fakeModel{reply: func(messages []ollama.Message, tools []ollama.Tool) (ollama.Response, error) {
		if len(tools) > 0 && messages[len(messages)-1].Role != "tool" {
			return ollama.Response{Message: ollama.Message{Calls: []ollama.Call{{Function: ollama.Function{Name: "search_documents", Arguments: []byte(`{"query":"question"}`)}}}}}, nil
		}
		return ollama.Response{Message: ollama.Message{Content: "Ответ"}}, nil
	}}
	agent := testAgent(t, model, rag)
	for _, team := range []string{"backend", "platform"} {
		_, err := agent.Generate(t.Context(), users.User{ID: team, Team: team}, []buddy.Message{{Role: "user", Content: "Вопрос"}}, nil)
		if err != nil || rag.team != team {
			t.Fatalf("team %s: got %s, %v", team, rag.team, err)
		}
	}
}

func TestGenerateSearchAttachStream(t *testing.T) {
	model := &fakeModel{}
	model.reply = func(messages []ollama.Message, tools []ollama.Tool) (ollama.Response, error) {
		switch model.calls {
		case 1:
			return ollama.Response{Message: ollama.Message{Content: "Ищу. ", Calls: []ollama.Call{{Function: ollama.Function{Name: "search_documents", Arguments: []byte(`{"query":"question"}`)}}}}}, nil
		case 2:
			if last := messages[len(messages)-1]; last.ToolName != "search_documents" || !strings.Contains(last.Content, `"title":"Title"`) {
				t.Fatal(last)
			}
			return ollama.Response{Message: ollama.Message{Content: "Нашёл. ", Calls: []ollama.Call{{Function: ollama.Function{Name: "attach_source", Arguments: []byte(`{"document_id":"doc","ordinal":0}`)}}}}}, nil
		case 3:
			if last := messages[len(messages)-1]; last.ToolName != "attach_source" || last.Content != `{"attached":true}` {
				t.Fatal(last)
			}
			return ollama.Response{Message: ollama.Message{Content: "Ответ"}}, nil
		default:
			t.Fatal("answer regenerated")
			return ollama.Response{}, nil
		}
	}
	var text string
	var states []string
	answer, err := testAgent(t, model, &fakeRAG{}).Generate(t.Context(), users.User{}, nil, func(event buddy.Event) error {
		text += event.Text
		if event.Type == "state" {
			states = append(states, event.State+":"+event.Progress)
		}
		return nil
	})
	if err != nil || answer.Content != "Ищу. Нашёл. Ответ" || answer.Content != text || answer.Source == nil || answer.Source.Title != "Title" || answer.Source.Quote != "Документ" || model.calls != 3 {
		t.Fatalf("%+v, %v", answer, err)
	}
	if strings.Join(states, ",") != "thinking:,answering:,progress:search_documents,thinking:,answering:,progress:attach_source,thinking:,answering:" {
		t.Fatal(states)
	}
}

func TestGenerateLimitAndCancellation(t *testing.T) {
	for _, overflow := range []bool{false, true} {
		t.Run(map[bool]string{false: "limit", true: "overflow"}[overflow], func(t *testing.T) {
			model := &fakeModel{}
			model.reply = func(_ []ollama.Message, tools []ollama.Tool) (ollama.Response, error) {
				if model.calls == 1 {
					return ollama.Response{Message: ollama.Message{Calls: []ollama.Call{{Function: ollama.Function{Name: "search_documents", Arguments: []byte(`{"query":"q"}`)}}}}}, nil
				}
				if model.calls == 2 {
					calls := make([]ollama.Call, 5)
					if overflow {
						calls = append(calls, ollama.Call{})
					}
					for i := range calls {
						calls[i] = ollama.Call{Function: ollama.Function{Name: "attach_source", Arguments: []byte(`{"document_id":"doc","ordinal":0}`)}}
					}
					return ollama.Response{Message: ollama.Message{Calls: calls}}, nil
				}
				if len(tools) != 0 {
					t.Fatal("tools available after limit")
				}
				return ollama.Response{Message: ollama.Message{Content: "Answer"}}, nil
			}
			var attaches int
			answer, err := testAgent(t, model, &fakeRAG{}).Generate(t.Context(), users.User{}, nil, func(e buddy.Event) error {
				if e.Progress == "attach_source" {
					attaches++
				}
				return nil
			})
			if overflow {
				if err == nil || attaches != 0 {
					t.Fatalf("overflow: %v, %d", err, attaches)
				}
				return
			}
			if err != nil || model.calls != 3 || attaches != 5 || answer.Source == nil {
				t.Fatalf("limit: %+v %v %d", answer, err, attaches)
			}
		})
	}
	for _, state := range []string{"thinking", "answering", "text", "cancel"} {
		t.Run(state, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			failure := errors.New("emit failed")
			model := &fakeModel{reply: func(_ []ollama.Message, _ []ollama.Tool) (ollama.Response, error) {
				return ollama.Response{Message: ollama.Message{Content: "partial"}}, nil
			}}
			if state == "cancel" {
				cancel()
			}
			_, err := testAgent(t, model, &fakeRAG{}).Generate(ctx, users.User{}, nil, func(e buddy.Event) error {
				if e.State == state || e.Type == state {
					return failure
				}
				return nil
			})
			if state == "cancel" {
				if !errors.Is(err, context.Canceled) || model.calls != 0 {
					t.Fatal(err)
				}
			} else if !errors.Is(err, failure) {
				t.Fatal(err)
			}
		})
	}
}

func TestGenerateSourceDoesNotLeak(t *testing.T) {
	model := &fakeModel{}
	model.reply = func(_ []ollama.Message, _ []ollama.Tool) (ollama.Response, error) {
		switch model.calls {
		case 1:
			return ollama.Response{Message: ollama.Message{Calls: []ollama.Call{{Function: ollama.Function{Name: "search_documents", Arguments: []byte(`{"query":"q"}`)}}}}}, nil
		case 2:
			return ollama.Response{Message: ollama.Message{Calls: []ollama.Call{{Function: ollama.Function{Name: "attach_source", Arguments: []byte(`{"document_id":"doc","ordinal":0}`)}}}}}, nil
		default:
			return ollama.Response{Message: ollama.Message{Content: "Answer"}}, nil
		}
	}
	agent := testAgent(t, model, &fakeRAG{})
	first, err := agent.Generate(t.Context(), users.User{ID: "first", Team: "backend"}, nil, nil)
	if err != nil || first.Source == nil {
		t.Fatalf("first: %+v %v", first, err)
	}
	second, err := agent.Generate(t.Context(), users.User{ID: "second", Team: "platform"}, nil, nil)
	if err != nil || second.Source != nil {
		t.Fatalf("second: %+v %v", second, err)
	}
}
