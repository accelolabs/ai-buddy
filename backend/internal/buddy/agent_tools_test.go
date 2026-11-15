package buddy

import (
	"ai-buddy/internal/ollama"
	"context"
	"encoding/json"
	"testing"
)

func TestRejectedToolArgumentsPreserveErrors(t *testing.T) {
	for _, tc := range []struct{ name, args, want string }{
		{"known", "{", "invalid tool call"},
		{"known", "", "invalid tool call"},
		{"unknown", "{}", "invalid tool call"},
		{"known", "null", "invalid tool arguments"},
		{"known", "[]", "invalid tool arguments"},
		{"known", "1", "invalid tool arguments"},
		{"known", "true", "invalid tool arguments"},
		{"known", "{} {}", "invalid tool call"},
	} {
		a := &Agent{}

		_, err := a.executeTools(
			context.Background(),
			Run{},
			nil,
			map[string]bool{"known": true},
			nil,
			[]ollama.Call{
				{
					Function: ollama.Function{
						Name:      tc.name,
						Arguments: json.RawMessage(tc.args),
					},
				},
			},
		)
		if err == nil || err.Error() != tc.want {
			t.Fatalf("%q: %v want %s", tc.args, err, tc.want)
		}
	}
}
