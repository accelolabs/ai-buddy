package buddy

import (
	"ai-buddy/internal/ollama"
	"ai-buddy/internal/users"
	"context"
	"errors"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Model interface {
	Chat(
		context.Context,
		[]ollama.Message,
		[]ollama.Tool,
		func(string) error,
	) (ollama.Response, error)
}
type Agent struct {
	Repo         Repository
	Tools        *Tools
	Model        Model
	gate         chan struct{}
	systemPrompt string
}

func NewAgent(repo Repository, tools *Tools, model Model, systemPrompt string) *Agent {
	return &Agent{
		Repo:         repo,
		Tools:        tools,
		Model:        model,
		gate:         make(chan struct{}, 1),
		systemPrompt: systemPrompt,
	}
}

func (a *Agent) Generate(ctx context.Context, u users.User, run Run) error {
	if e := a.Repo.Event(ctx, run.ID, EventStage, "waiting_for_model"); e != nil {
		return e
	}

	select {
	case a.gate <- struct{}{}:
		defer func() { <-a.gate }()
	case <-ctx.Done():
		return ctx.Err()
	}

	messages, request, e := a.context(ctx, u, run)
	if e != nil {
		return e
	}

	session, e := a.Tools.Open(ctx, u, run, request)
	if e != nil {
		return e
	}
	defer session.Close()

	listed, e := session.Client.ListTools(ctx, &sdk.ListToolsParams{})
	if e != nil {
		return e
	}

	tools := []ollama.Tool{}
	known := map[string]bool{}

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
	for calls != 6 {
		if e := a.Repo.Event(ctx, run.ID, EventStage, "choosing_tools"); e != nil {
			return e
		}

		messages, e = Budget(messages, tools)
		if e != nil {
			return e
		}

		r, err := a.Model.Chat(ctx, messages, tools, nil)
		if err != nil {
			return err
		}

		if len(r.Message.Calls) == 0 {
			break
		}

		if calls+len(r.Message.Calls) > 6 {
			return errors.New("tool call limit exceeded")
		}

		r.Message.Role = roleAssistant
		messages = append(messages, r.Message)

		messages, e = a.executeTools(ctx, run, session, known, messages, r.Message.Calls)
		if e != nil {
			return e
		}

		calls += len(r.Message.Calls)
	}

	return a.answer(ctx, run, messages)
}
