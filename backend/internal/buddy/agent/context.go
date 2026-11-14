package agent

import (
	"context"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/buddy/ollama"
	"ai-buddy/internal/platform/auth"
)

func (a *Agent) context(ctx context.Context, u auth.User, run buddy.Run) ([]ollama.Message, string, error) {
	h, e := a.Repo.History(ctx, u.ID, run.ChatID)
	if e != nil {
		return nil, "", e
	}
	var request string
	messages := []ollama.Message{{Role: "system", Content: system}}
	// Include only completed exchanges; retries use the same original user message.

	byID := map[string]buddy.Message{}
	answers := map[string]buddy.Message{}
	for _, m := range h.Messages {
		byID[m.ID] = m
		if m.Role == "assistant" {
			answers[m.RunID] = m
		}
	}
	request = byID[run.MessageID].Content
	for _, r := range h.Runs {
		if r.Status == buddy.RunDone && r.MessageID != run.MessageID {
			user, ok := byID[r.MessageID]
			answer, has := answers[r.ID]
			if ok && has {
				messages = append(messages, ollama.Message{Role: "user", Content: user.Content}, ollama.Message{Role: "assistant", Content: answer.Content})
			}
		}
	}
	messages = append(messages, ollama.Message{Role: "user", Content: request})

	return messages, request, nil
}
