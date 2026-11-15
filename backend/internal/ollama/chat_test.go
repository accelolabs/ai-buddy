package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChat(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request chatRequest
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				if r.URL.Path != "/api/chat" || request.Model != "qwen3.5:4b" || request.Stream != stream || request.Think || request.Options.Context != 16384 || request.Options.Predict != 512 || len(request.Messages) != 1 || len(request.Tools) != 1 {
					t.Errorf("request: %+v", request)
				}
				if stream {
					_, _ = fmt.Fprintln(w, `{"message":{"role":"assistant","content":"При"},"done":false}`)
					_, _ = fmt.Fprintln(w, `{"message":{"role":"assistant","content":"вет"},"done":false}`)
					_, _ = fmt.Fprintln(w, `{"message":{"role":"assistant"},"done":true}`)
				} else {
					_, _ = fmt.Fprintln(w, `{"message":{"role":"assistant","tool_calls":[{"function":{"name":"search","arguments":{"query":"q"}}}]},"done":true}`)
				}
			}))
			defer server.Close()
			var emit func(string) error
			var text string
			if stream {
				emit = func(part string) error { text += part; return nil }
			}
			result, err := NewChatClient(server.URL, "qwen3.5:4b", 16384).Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, []Tool{{Type: "function"}}, emit)
			if err != nil {
				t.Fatal(err)
			}
			if stream {
				if text != "Привет" || result.Message.Content != text {
					t.Fatalf("result: %+v, text: %s", result, text)
				}
			} else if len(result.Message.Calls) != 1 || result.Message.Calls[0].Function.Name != "search" {
				t.Fatalf("result: %+v", result)
			}
		})
	}
}

func TestChatErrors(t *testing.T) {
	for _, body := range []string{`broken`, `{"error":"failed"}`, `{"message":{"content":"partial"},"done":false}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprintln(w, body) }))
		_, err := NewChatClient(server.URL, "model", 8192).Chat(t.Context(), nil, nil, func(string) error { return nil })
		server.Close()
		if err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintln(w, `{"message":{"content":"text"},"done":true}`)
	}))
	defer server.Close()
	failure := errors.New("consumer stopped")
	_, err := NewChatClient(server.URL, "model", 8192).Chat(t.Context(), nil, nil, func(string) error { return failure })
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := NewChatClient(server.URL, "model", 8192).Chat(ctx, nil, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
