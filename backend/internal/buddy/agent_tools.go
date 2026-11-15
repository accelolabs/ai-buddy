package buddy

import (
	"ai-buddy/internal/ollama"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (a *Agent) executeTools(
	ctx context.Context,
	run Run,
	session *Session,
	known map[string]bool,
	messages []ollama.Message,
	calls []ollama.Call,
) ([]ollama.Message, error) {
	for _, call := range calls {
		if !known[call.Function.Name] {
			return nil, errors.New("invalid tool call")
		}

		var args map[string]any

		err := json.Unmarshal(call.Function.Arguments, &args)

		if syntaxError, ok := errors.AsType[*json.SyntaxError](
			err,
		); ok &&
			syntaxError != nil {
			return nil, errors.New("invalid tool call")
		}

		if err != nil || args == nil {
			return nil, errors.New("invalid tool arguments")
		}

		if e := a.Repo.Event(ctx, run.ID, EventStage, call.Function.Name); e != nil {
			return nil, e
		}

		result, err := session.Client.CallTool(
			ctx,
			&sdk.CallToolParams{Name: call.Function.Name, Arguments: args},
		)
		if err != nil {
			return nil, fmt.Errorf("tool failed: %w", err)
		}

		var text strings.Builder

		for _, content := range result.Content {
			if v, ok := content.(*sdk.TextContent); ok {
				_, _ = text.WriteString(v.Text)
			}
		}

		if result.IsError {
			return nil, fmt.Errorf("tool failed: %s", clip(text.String(), 300))
		}

		messages = append(
			messages,
			ollama.Message{
				Role:     roleTool,
				ToolName: call.Function.Name,
				Content:  clip(text.String(), 2400),
			},
		)
	}

	return messages, nil
}
