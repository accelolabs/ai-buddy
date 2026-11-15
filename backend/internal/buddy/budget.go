package buddy

import (
	"encoding/json"
	"errors"

	"ai-buddy/internal/ollama"
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
		idx := -1
		for i := 1; i < len(messages)-1; i++ {
			if messages[i].Role == "user" && messages[i+1].Role == "assistant" && len(messages[i+1].Calls) == 0 {
				idx = i
				break
			}
		}
		if idx >= 0 {
			messages = append(messages[:idx:idx], messages[idx+2:]...)
			continue
		}
		changed := false
		for i := range messages {
			if messages[i].Role == "tool" && len(messages[i].Content) > 128 {
				messages[i].Content = clip(messages[i].Content, len(messages[i].Content)/2)
				changed = true
				break
			}
		}
		if !changed {
			// Discard an old complete tool round before the latest one.
			removed := false
			for i := 1; i < len(messages)-1; i++ {
				if messages[i].Role == "assistant" && len(messages[i].Calls) > 0 {
					end := i + 1
					for end < len(messages) && messages[end].Role == "tool" {
						end++
					}
					if end < len(messages) {
						messages = append(messages[:i:i], messages[end:]...)
						removed = true
					}
					break
				}
			}
			if !removed {
				return nil, errors.New("request exceeds context budget; use a shorter message")
			}
		}
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
