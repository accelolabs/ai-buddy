package buddy

import (
	"backend-ai/internal/ollama"
	"encoding/json"
	"errors"
)

// Budget counts UTF-8 bytes conservatively (at most one token per byte), including
// serialized tool schemas, protocol envelopes and a 512-token response reserve.
// Oversized current inputs are rejected instead of silently losing instructions.
func Budget(messages []ollama.Message, tools []ollama.Tool) ([]ollama.Message, error) {
	cost := func() (int, error) {
		b, err := json.Marshal(struct {
			M []ollama.Message
			T []ollama.Tool
		}{messages, tools})

		return len(b) + 512 + 128, err
	}

	for {
		size, err := cost()
		if err != nil {
			return nil, err
		}

		if size <= 4096 {
			break
		}

		if shorter, removed := dropConversationTurn(messages); removed {
			messages = shorter
			continue
		}

		if shrinkToolResult(messages) {
			continue
		}

		shorter, removed := dropToolRound(messages)
		if !removed {
			return nil, errors.New(
				"request exceeds context budget; use a shorter message",
			)
		}

		messages = shorter
	}

	return messages, nil
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}

	for n > 0 && n < len(s) && s[n]&0xc0 == 0x80 {
		n--
	}

	return s[:n] + "…"
}

func dropConversationTurn(messages []ollama.Message) ([]ollama.Message, bool) {
	for i := 1; i < len(messages)-1; i++ {
		if messages[i].Role == roleUser && messages[i+1].Role == roleAssistant &&
			len(messages[i+1].Calls) == 0 {
			return append(messages[:i:i], messages[i+2:]...), true
		}
	}

	return messages, false
}

func shrinkToolResult(messages []ollama.Message) bool {
	for i := range messages {
		if messages[i].Role == roleTool && len(messages[i].Content) > 128 {
			messages[i].Content = clip(messages[i].Content, len(messages[i].Content)/2)
			return true
		}
	}

	return false
}

// dropToolRound retains the latest tool round and the current request.
func dropToolRound(messages []ollama.Message) ([]ollama.Message, bool) {
	for i := 1; i < len(messages)-1; i++ {
		if messages[i].Role != roleAssistant || len(messages[i].Calls) == 0 {
			continue
		}

		end := i + 1
		for end < len(messages) && messages[end].Role == roleTool {
			end++
		}

		if end < len(messages) {
			return append(messages[:i:i], messages[end:]...), true
		}

		break
	}

	return messages, false
}
