package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	// Project packages.
	"backend/internal/buddy"
	buddymcp "backend/internal/buddy/mcp"
	"backend/internal/ollama"
	"backend/internal/tracker"
	"backend/internal/users"
	"backend/internal/wiki/rag"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxToolCalls = 6

type model interface {
	Chat(context.Context, []ollama.Message, []ollama.Tool, func(string) error) (ollama.Response, error)
}

type searcher interface {
	Search(context.Context, string, string) ([]rag.Match, error)
}

type mentorService interface {
	GetMentor(context.Context, string) (users.User, error)
}

type taskService interface {
	List(context.Context, users.User) ([]tracker.Task, error)
}

type Agent struct {
	tracker      taskService
	users        mentorService
	rag          searcher
	model        model
	systemPrompt string
}

func NewAgent(model model, rag searcher, promptFile string, users mentorService, tracker taskService) (*Agent, error) {
	prompt, err := os.ReadFile(promptFile)
	if err != nil {
		return nil, fmt.Errorf("read system prompt: %w", err)
	}

	return &Agent{
		users:        users,
		tracker:      tracker,
		model:        model,
		rag:          rag,
		systemPrompt: string(prompt),
	}, nil
}

// Generate creates and closes an MCP session bound to the trusted current user.
func (a *Agent) Generate(
	ctx context.Context,
	user users.User,
	history []buddy.Message,
	emit func(buddy.Event) error,
) (buddy.Answer, error) {
	handler := buddymcp.NewHandler(a.rag, user, a.users, a.tracker)
	session, closeSession, err := handler.OpenSession(ctx)
	if err != nil {
		return buddy.Answer{}, err
	}

	defer closeSession()

	messages := make([]ollama.Message, 0, len(history)+1)
	messages = append(messages, ollama.Message{
		Role:    "system",
		Content: a.systemPrompt,
	})

	for _, message := range history {
		messages = append(messages, ollama.Message{
			Role:    message.Role,
			Content: message.Content,
		})
	}

	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		return buddy.Answer{}, fmt.Errorf("list tools: %w", err)
	}

	tools := make([]ollama.Tool, 0, len(listed.Tools))
	known := make(map[string]bool)

	for _, tool := range listed.Tools {
		tools = append(
			tools,
			ollama.Tool{
				Type: "function",
				Function: ollama.Definition{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.InputSchema,
				},
			},
		)
		known[tool.Name] = true
	}

	calls := 0

	var content strings.Builder

	for {
		if err := ctx.Err(); err != nil {
			return buddy.Answer{}, err
		}

		if emit != nil {
			if err := emit(buddy.Event{
				Type:  "state",
				State: "thinking",
			}); err != nil {
				return buddy.Answer{}, err
			}
		}

		if calls == maxToolCalls {
			tools = nil
		}

		response, err := a.streamResponse(ctx, messages, tools, &content, emit)
		if err != nil {
			return buddy.Answer{}, fmt.Errorf("generate answer: %w", err)
		}

		if len(response.Message.Calls) == 0 {
			return buddy.Answer{
				Content:        content.String(),
				Source:         handler.Source(),
				Ticket:         handler.Ticket(),
				Task:           handler.Task(),
				Tasks:          handler.Tasks(),
				TaskCompletion: handler.TaskCompletion(),
			}, nil
		}

		if calls+len(response.Message.Calls) > maxToolCalls {
			return buddy.Answer{}, fmt.Errorf("tool call limit exceeded")
		}

		response.Message.Role = "assistant"
		messages = append(messages, response.Message)
		results, err := a.callTools(ctx, session, response.Message.Calls, known, emit)
		if err != nil {
			return buddy.Answer{}, err
		}

		messages = append(messages, results...)
		calls += len(response.Message.Calls)
	}
}

func (a *Agent) streamResponse(
	ctx context.Context,
	messages []ollama.Message,
	tools []ollama.Tool,
	content *strings.Builder,
	emit func(buddy.Event) error,
) (ollama.Response, error) {
	answering := false

	return a.model.Chat(ctx, messages, tools, func(text string) error {
		if text == "" {
			return nil
		}

		if !answering {
			if emit != nil {
				if err := emit(buddy.Event{
					Type:  "state",
					State: "answering",
				}); err != nil {
					return err
				}
			}

			answering = true
		}

		content.WriteString(text)
		if emit != nil {
			return emit(buddy.Event{
				Type: "text",
				Text: text,
			})
		}

		return nil
	})
}

func (a *Agent) callTools(
	ctx context.Context,
	session *sdk.ClientSession,
	calls []ollama.Call,
	known map[string]bool,
	emit func(buddy.Event) error,
) ([]ollama.Message, error) {
	messages := make([]ollama.Message, 0, len(calls))

	for _, call := range calls {
		if !known[call.Function.Name] {
			messages = append(messages, toolError(call.Function.Name, "unknown tool"))
			continue
		}

		var args map[string]any
		if err := json.Unmarshal(call.Function.Arguments, &args); err != nil || args == nil {
			messages = append(messages, toolError(call.Function.Name, "invalid arguments"))
			continue
		}

		if emit != nil {
			if err := emit(buddy.Event{
				Type:     "state",
				State:    "progress",
				Progress: call.Function.Name,
			}); err != nil {
				return nil, err
			}
		}

		result, err := session.CallTool(ctx, &sdk.CallToolParams{
			Name:      call.Function.Name,
			Arguments: args,
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}

			messages = append(messages, toolError(call.Function.Name, err.Error()))
			continue
		}

		var text strings.Builder

		for _, content := range result.Content {
			if part, ok := content.(*sdk.TextContent); ok {
				text.WriteString(part.Text)
			}
		}

		if result.IsError {
			messages = append(messages, toolError(call.Function.Name, text.String()))
			continue
		}

		messages = append(messages, ollama.Message{
			Role:     "tool",
			ToolName: call.Function.Name,
			Content:  text.String(),
		})
	}

	return messages, nil
}

func toolError(name, detail string) ollama.Message {
	log.Printf("buddy tool %q: %s", name, detail)

	content, _ := json.Marshal(map[string]string{"error": "Сбой инструмента. Действие не выполнено."})

	return ollama.Message{
		Role:     "tool",
		ToolName: name,
		Content:  string(content),
	}
}
