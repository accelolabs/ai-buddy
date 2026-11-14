package agent

import (
	"context"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/buddy/ollama"
)

func (a *Agent) answer(ctx context.Context, run buddy.Run, messages []ollama.Message) error {
	if e := a.Repo.Event(ctx, run.ID, buddy.EventStage, "answering"); e != nil {
		return e
	}
	messages, e := Budget(messages, nil)
	if e != nil {
		return e
	}
	r, e := a.Model.Chat(ctx, messages, nil, true, func(text string) error { return a.Repo.Event(ctx, run.ID, buddy.EventText, text) })
	if e != nil {
		return e
	}
	return a.Repo.Event(ctx, run.ID, buddy.EventUsage, map[string]int{"prompt_eval_count": r.PromptTokens, "num_ctx": 4096})
}
