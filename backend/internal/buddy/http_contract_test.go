package buddy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestProjectionKeys(t *testing.T) {
	for _, tc := range []struct {
		value any
		keys  []string
	}{
		{Chat{Owner: "private"}, []string{"id", "title", "created_at"}},
		{Action{Owner: "private"}, []string{"id", "run_id", "kind", "status", "fields", "result", "error"}},
		{Message{ChatID: "private"}, []string{"id", "run_id", "role", "content", "created_at"}},
		{Card{}, []string{"run_id", "kind", "data"}},
	} {
		data, err := json.Marshal(tc.value)
		if err != nil {
			t.Fatal(err)
		}

		var object map[string]json.RawMessage
		if err = json.Unmarshal(data, &object); err != nil {
			t.Fatal(err)
		}

		keys := make([]string, 0, len(object))
		for k := range object {
			keys = append(keys, k)
		}

		slices.Sort(keys)
		slices.Sort(tc.keys)

		if !reflect.DeepEqual(keys, tc.keys) {
			t.Fatalf("keys %v want %v", keys, tc.keys)
		}
	}
}

type emptyRepository struct{ Repository }

func (emptyRepository) Chats(context.Context, string) ([]Chat, error) { return []Chat{}, nil }
func (emptyRepository) History(context.Context, string, string) (History, error) {
	return History{
		Messages: []Message{},
		Runs:     []Run{},
		Actions:  []Action{},
		Cards:    []Card{},
	}, nil
}

func TestEmptyResponseBoundaries(t *testing.T) {
	h := NewHandler(&Service{Repo: emptyRepository{}})
	for _, tc := range []struct {
		handler http.HandlerFunc
		arrays  []string
	}{
		{h.list, []string{"chats"}}, {h.history, []string{"messages", "runs", "actions", "cards"}},
	} {
		w := httptest.NewRecorder()
		tc.handler(
			w,
			httptest.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				"/",
				nil,
			),
		)

		var object map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &object); err != nil {
			t.Fatal(err)
		}

		for _, key := range tc.arrays {
			if string(object[key]) != "[]" {
				t.Fatal(key, w.Body)
			}
		}

		if latest, ok := object["latest_run"]; ok && string(latest) != "null" {
			t.Fatal(w.Body)
		}
	}

	raw, err := json.Marshal(
		Action{Fields: json.RawMessage(`{"title":"T"}`), Result: nil, Owner: "private"},
	)
	if err != nil || !strings.Contains(string(raw), `"result":null`) ||
		strings.Contains(string(raw), "private") {
		t.Fatal(string(raw), err)
	}
}
