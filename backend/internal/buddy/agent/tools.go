package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/buddy/mcp"
	"ai-buddy/internal/buddy/ollama"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (a *Agent) executeTools(ctx context.Context, run buddy.Run, session *mcp.Session, known map[string]bool, messages []ollama.Message, calls []ollama.Call) ([]ollama.Message, error) {
	for _, call := range calls {
		if !known[call.Function.Name] || !json.Valid(call.Function.Arguments) {
			return nil, errors.New("invalid tool call")
		}
		var args map[string]any
		if json.Unmarshal(call.Function.Arguments, &args) != nil || args == nil {
			return nil, errors.New("invalid tool arguments")
		}
		if e := a.Repo.Event(ctx, run.ID, buddy.EventStage, call.Function.Name); e != nil {
			return nil, e
		}
		result, err := session.Client.CallTool(ctx, &sdk.CallToolParams{Name: call.Function.Name, Arguments: args})
		if err != nil {
			return nil, fmt.Errorf("tool failed: %w", err)
		}
		var text strings.Builder
		for _, content := range result.Content {
			if v, ok := content.(*sdk.TextContent); ok {
				text.WriteString(v.Text)
			}
		}
		if result.IsError {
			return nil, fmt.Errorf("tool failed: %s", clip(text.String(), 300))
		}
		messages = append(messages, ollama.Message{Role: "tool", ToolName: call.Function.Name, Content: clip(text.String(), 2400)})
	}
	return messages, nil
}
