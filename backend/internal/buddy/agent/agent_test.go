package agent

import (
	"backend/internal/buddy"
	"backend/internal/ollama"
	"backend/internal/users"
	"backend/internal/wiki"
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
	if err == nil && emit != nil {
		err = emit(response.Message.Content)
	}
	return response, err
}

type fakeRAG struct{ query, team string }

func (r *fakeRAG) Search(_ context.Context, query, team string) ([]wiki.Chunk, error) {
	r.query, r.team = query, team
	return []wiki.Chunk{{DocumentID: "doc", Text: "Документ"}}, nil
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
			if len(tools) != 1 || tools[0].Function.Name != "search_documents" {
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
	history := make([]ollama.Message, 1, 10)
	history[0] = ollama.Message{Role: "user", Content: "Вопрос"}
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
	if err != nil || answer != "Ответ" || emitted != answer || rag.team != "backend" || rag.query != "question" {
		t.Fatalf("answer: %s, %v", answer, err)
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
		_, err := agent.Generate(t.Context(), users.User{ID: team, Team: team}, []ollama.Message{{Role: "user", Content: "Вопрос"}}, nil)
		if err != nil || rag.team != team {
			t.Fatalf("team %s: got %s, %v", team, rag.team, err)
		}
	}
}
