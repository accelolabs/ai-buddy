package buddy

import (
	"backend-ai/internal/ollama"
	"context"
)

func (a *Agent) answer(ctx context.Context, run Run, messages []ollama.Message) error {
	if e := a.Repo.Event(ctx, run.ID, EventStage, "answering"); e != nil {
		return e
	}

	messages, e := Budget(messages, nil)
	if e != nil {
		return e
	}

	r, e := a.Model.Chat(
		ctx,
		messages,
		nil,
		func(text string) error { return a.Repo.Event(ctx, run.ID, EventText, text) },
	)
	if e != nil {
		return e
	}

	return a.Repo.Event(
		ctx,
		run.ID,
		EventUsage,
		map[string]int{"prompt_eval_count": r.PromptTokens, "num_ctx": 4096},
	)
}
