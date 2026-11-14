package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNativeChat(t *testing.T) {
	ctx := context.Background()
	mode := "stream"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Error(r.URL)
		}
		var req Request
		if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
			t.Error(e)
		}
		if req.Model != "qwen3.5:4b" || req.Think || req.Options["num_ctx"] != 4096 || req.Options["num_predict"] != 512 {
			t.Error(req)
		}
		switch mode {
		case "stream":
			fmt.Fprintln(w, `{"message":{"role":"assistant","content":"hello "},"done":false}`)
			fmt.Fprintln(w, `{"message":{"content":"world"},"done":true,"prompt_eval_count":123}`)
		case "tool":
			fmt.Fprintln(w, `{"message":{"role":"assistant","tool_calls":[{"function":{"name":"get_progress","arguments":{}}}]},"done":true}`)
		case "http":
			w.WriteHeader(503)
		case "incomplete":
			fmt.Fprintln(w, `{"message":{"content":"partial"}}`)
		case "error":
			fmt.Fprintln(w, `{"error":"failed"}`)
		case "bad":
			fmt.Fprintln(w, `no json`)
		}
	}))
	defer server.Close()
	c := New(server.URL)
	var out strings.Builder
	r, e := c.Chat(ctx, []Message{{Role: "user", Content: "hi"}}, nil, true, func(s string) error { out.WriteString(s); return nil })
	if e != nil || out.String() != "hello world" || r.PromptTokens != 123 {
		t.Fatal(r, e, out.String())
	}
	mode = "tool"
	r, e = c.Chat(ctx, nil, []Tool{{Type: "function", Function: Definition{Name: "get_progress", Parameters: map[string]any{"type": "object"}}}}, false, nil)
	if e != nil || len(r.Message.Calls) != 1 {
		t.Fatal(r, e)
	}
	if _, e = c.Chat(ctx, nil, nil, true, func(string) error { return nil }); e == nil {
		t.Fatal("stream tool")
	}
	for _, v := range []string{"http", "incomplete", "error", "bad"} {
		mode = v
		if _, e = c.Chat(ctx, nil, nil, true, func(string) error { return nil }); e == nil {
			t.Fatal(v)
		}
	}
	mode = "stream"
	if _, e = c.Chat(ctx, nil, nil, true, func(string) error { return errors.New("save failed") }); e == nil {
		t.Fatal("callback")
	}
	c.Base = ":"
	if _, e = c.Chat(ctx, nil, nil, false, nil); e == nil {
		t.Fatal("URL")
	}
	c.Base = server.URL
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, e = c.Chat(cancelled, nil, nil, false, nil); e == nil {
		t.Fatal("cancel")
	}
}
