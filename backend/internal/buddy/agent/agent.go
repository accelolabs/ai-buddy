package agent

import (
	"backend/internal/buddy"
	buddymcp "backend/internal/buddy/mcp"
	"backend/internal/ollama"
	"backend/internal/users"
	"backend/internal/wiki/rag"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxToolCalls = 6

type model interface {
	Chat(context.Context, []ollama.Message, []ollama.Tool, func(string) error) (ollama.Response, error)
}

type searcher interface {
	Search(context.Context, string, string) ([]rag.Match, error)
}

type Agent struct {
	rag          searcher
	model        model
	systemPrompt string
}

func NewAgent(model model, rag searcher, promptFile string) (*Agent, error) {
	prompt, err := os.ReadFile(promptFile)
	if err != nil {
		return nil, fmt.Errorf("read system prompt: %w", err)
	}
	return &Agent{model: model, rag: rag, systemPrompt: string(prompt)}, nil
}

// Generate creates and closes an MCP session bound to the trusted current user.
func (a *Agent) Generate(ctx context.Context, user users.User, history []buddy.Message, emit func(buddy.Event) error) (buddy.Answer, error) {
	handler := buddymcp.NewHandler(a.rag, user.Team)
	session, closeSession, err := handler.OpenSession(ctx)
	if err != nil {
		return buddy.Answer{}, err
	}
	defer closeSession()

	messages := make([]ollama.Message, 0, len(history)+1)
	messages = append(messages, ollama.Message{Role: "system", Content: a.systemPrompt})
	for _, message := range history {
		messages = append(messages, ollama.Message{Role: message.Role, Content: message.Content})
	}
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		return buddy.Answer{}, fmt.Errorf("list tools: %w", err)
	}
	tools := make([]ollama.Tool, 0, len(listed.Tools))
	known := make(map[string]bool)
	for _, tool := range listed.Tools {
		tools = append(tools, ollama.Tool{Type: "function", Function: ollama.Definition{Name: tool.Name, Description: tool.Description, Parameters: tool.InputSchema}})
		known[tool.Name] = true
	}
	sendState := func(state, progress string) error {
		if emit == nil {
			return nil
		}
		return emit(buddy.Event{Type: "state", State: state, Progress: progress})
	}
	calls := 0
	var content strings.Builder
	for {
		if err := ctx.Err(); err != nil {
			return buddy.Answer{}, err
		}
		if err := sendState("thinking", ""); err != nil {
			return buddy.Answer{}, err
		}
		if calls == maxToolCalls {
			tools = nil
		}
		answering := false
		response, err := a.model.Chat(ctx, messages, tools, func(text string) error {
			if text == "" {
				return nil
			}
			if !answering {
				if err := sendState("answering", ""); err != nil {
					return err
				}
				answering = true
			}
			content.WriteString(text)
			if emit != nil {
				return emit(buddy.Event{Type: "text", Text: text})
			}
			return nil
		})
		if err != nil {
			return buddy.Answer{}, fmt.Errorf("generate answer: %w", err)
		}
		if len(response.Message.Calls) == 0 {
			return buddy.Answer{Content: content.String(), Source: handler.Source()}, nil
		}
		if calls+len(response.Message.Calls) > maxToolCalls {
			return buddy.Answer{}, fmt.Errorf("tool call limit exceeded")
		}
		response.Message.Role = "assistant"
		messages = append(messages, response.Message)
		for _, call := range response.Message.Calls {
			if !known[call.Function.Name] {
				return buddy.Answer{}, fmt.Errorf("unknown tool %q", call.Function.Name)
			}
			var args map[string]any
			if err := json.Unmarshal(call.Function.Arguments, &args); err != nil || args == nil {
				return buddy.Answer{}, fmt.Errorf("invalid arguments for tool %q", call.Function.Name)
			}
			if err := sendState("progress", call.Function.Name); err != nil {
				return buddy.Answer{}, err
			}
			result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: call.Function.Name, Arguments: args})
			if err != nil {
				return buddy.Answer{}, fmt.Errorf("call tool %q: %w", call.Function.Name, err)
			}
			var text strings.Builder
			for _, content := range result.Content {
				if part, ok := content.(*sdk.TextContent); ok {
					text.WriteString(part.Text)
				}
			}
			if result.IsError {
				return buddy.Answer{}, fmt.Errorf("tool %q failed: %s", call.Function.Name, text.String())
			}
			messages = append(messages, ollama.Message{Role: "tool", ToolName: call.Function.Name, Content: text.String()})
			calls++
		}
	}
}
