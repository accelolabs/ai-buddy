package buddy

import (
	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/ticket"
	"ai-buddy/internal/tracker"
	"bytes"
	"encoding/json"
	"strings"
	"time"
)

func DecodeFields(raw json.RawMessage, v any) error {
	if httpx.ReadJSON(bytes.NewReader(raw), v) != nil {
		return ErrInvalid
	}

	return nil
}

func parseFields(kind string, raw json.RawMessage) (any, error) {
	valid := func(s string, limit int) bool { return strings.TrimSpace(s) != "" && len([]rune(s)) <= limit }

	switch kind {
	case "create_task":
		var f tracker.CreateInput
		if DecodeFields(raw, &f) != nil || !valid(f.Title, 200) ||
			len([]rune(f.Description)) > 10000 {
			return nil, ErrInvalid
		}

		if _, err := time.Parse(time.DateOnly, f.Deadline); err != nil {
			return nil, ErrInvalid
		}

		return f, nil
	case "create_ticket":
		var f ticket.CreateInput
		if DecodeFields(raw, &f) != nil || !valid(f.Question, 200) ||
			!valid(f.Description, 10000) {
			return nil, ErrInvalid
		}

		return f, nil
	}

	return nil, ErrInvalid
}
