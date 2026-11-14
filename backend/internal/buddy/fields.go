package buddy

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"
)

type TaskFields struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Deadline    string `json:"deadline"`
}
type TicketFields struct {
	Question    string `json:"question"`
	Description string `json:"description"`
}

func DecodeFields(raw json.RawMessage, v any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || d.Decode(new(any)) != io.EOF {
		return ErrInvalid
	}
	return nil
}
func ValidateFields(kind string, raw json.RawMessage) error {
	valid := func(s string, max int) bool { return strings.TrimSpace(s) != "" && len([]rune(s)) <= max }
	switch kind {
	case "create_task":
		var f TaskFields
		if DecodeFields(raw, &f) != nil || !valid(f.Title, 200) || len([]rune(f.Description)) > 10000 {
			return ErrInvalid
		}
		if _, err := time.Parse("2006-01-02", f.Deadline); err != nil {
			return ErrInvalid
		}
	case "create_ticket":
		var f TicketFields
		if DecodeFields(raw, &f) != nil || !valid(f.Question, 200) || !valid(f.Description, 10000) {
			return ErrInvalid
		}
	default:
		return ErrInvalid
	}
	return nil
}
