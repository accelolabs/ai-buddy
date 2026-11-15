package buddy

import (
	"context"

	"ai-buddy/internal/ollama"
	"ai-buddy/internal/users"
)

func (a *Agent) context(ctx context.Context, u users.User, run Run) ([]ollama.Message, string, error) {
	h, e := a.Repo.History(ctx, u.ID, run.ChatID)
	if e != nil {
		return nil, "", e
	}
	var request string
	messages := []ollama.Message{{Role: "system", Content: system}}
	// Include only completed exchanges; retries use the same original user message.

	byID := map[string]Message{}
	answers := map[string]Message{}
	for _, m := range h.Messages {
		byID[m.ID] = m
		if m.Role == "assistant" {
			answers[m.RunID] = m
		}
	}
	request = byID[run.MessageID].Content
	for _, r := range h.Runs {
		if r.Status == RunDone && r.MessageID != run.MessageID {
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
