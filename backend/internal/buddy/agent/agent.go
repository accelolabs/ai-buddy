package agent

import (
	"backend/internal/buddy"
	"backend/internal/ollama"
	"backend/internal/users"
	"backend/internal/wiki"
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
	Search(context.Context, string, string) ([]wiki.Chunk, error)
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
func (a *Agent) Generate(ctx context.Context, user users.User, history []ollama.Message, emit func(buddy.Event) error) (string, error) {
	session, closeSession, err := a.openSession(ctx, user)
	if err != nil {
		return "", err
	}
	defer closeSession()

	messages := make([]ollama.Message, 0, len(history)+1)
	messages = append(messages, ollama.Message{Role: "system", Content: a.systemPrompt})
	messages = append(messages, history...)
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("list tools: %w", err)
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
	for len(tools) > 0 && calls < maxToolCalls {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if err := sendState("thinking", ""); err != nil {
			return "", err
		}
		response, err := a.model.Chat(ctx, messages, tools, nil)
		if err != nil {
			return "", fmt.Errorf("choose tools: %w", err)
		}
		if len(response.Message.Calls) == 0 {
			break
		}
		if calls+len(response.Message.Calls) > maxToolCalls {
			return "", fmt.Errorf("tool call limit exceeded")
		}
		response.Message.Role = "assistant"
		messages = append(messages, response.Message)
		for _, call := range response.Message.Calls {
			if !known[call.Function.Name] {
				return "", fmt.Errorf("unknown tool %q", call.Function.Name)
			}
			var args map[string]any
			if err := json.Unmarshal(call.Function.Arguments, &args); err != nil || args == nil {
				return "", fmt.Errorf("invalid arguments for tool %q", call.Function.Name)
			}
			if err := sendState("progress", call.Function.Name); err != nil {
				return "", err
			}
			result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: call.Function.Name, Arguments: args})
			if err != nil {
				return "", fmt.Errorf("call tool %q: %w", call.Function.Name, err)
			}
			var text strings.Builder
			for _, content := range result.Content {
				if part, ok := content.(*sdk.TextContent); ok {
					text.WriteString(part.Text)
				}
			}
			if result.IsError {
				return "", fmt.Errorf("tool %q failed: %s", call.Function.Name, text.String())
			}
			messages = append(messages, ollama.Message{Role: "tool", ToolName: call.Function.Name, Content: text.String()})
			calls++
		}
	}
	if err := sendState("answering", ""); err != nil {
		return "", err
	}
	var emitText func(string) error
	if emit != nil {
		emitText = func(text string) error { return emit(buddy.Event{Type: "text", Text: text}) }
	}
	response, err := a.model.Chat(ctx, messages, nil, emitText)
	if err != nil {
		return "", fmt.Errorf("generate answer: %w", err)
	}
	if len(response.Message.Calls) != 0 {
		return "", fmt.Errorf("unexpected tool call in final answer")
	}
	return response.Message.Content, nil
}
