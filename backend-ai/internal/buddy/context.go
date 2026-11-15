package buddy

import (
	"backend-ai/internal/ollama"
	"backend-ai/internal/users"
	"context"
)

func (a *Agent) context(
	ctx context.Context,
	u users.User,
	run Run,
) ([]ollama.Message, string, error) {
	h, e := a.Repo.History(ctx, u.ID, run.ChatID)
	if e != nil {
		return nil, "", e
	}

	var request string

	messages := []ollama.Message{{Role: "system", Content: a.systemPrompt}}
	// Include only completed exchanges; retries use the same original user message.

	byID := map[string]Message{}
	answers := map[string]Message{}

	for _, m := range h.Messages {
		byID[m.ID] = m
		if m.Role == roleAssistant {
			answers[m.RunID] = m
		}
	}

	request = byID[run.MessageID].Content

	for _, r := range h.Runs {
		if r.Status == RunDone && r.MessageID != run.MessageID {
			user, ok := byID[r.MessageID]

			answer, has := answers[r.ID]
			if ok && has {
				messages = append(
					messages,
					ollama.Message{Role: roleUser, Content: user.Content},
					ollama.Message{
						Role:    roleAssistant,
						Content: answer.Content,
					},
				)
			}
		}
	}

	messages = append(messages, ollama.Message{Role: roleUser, Content: request})

	return messages, request, nil
}
