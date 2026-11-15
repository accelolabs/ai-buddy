// Package ollama implements the chat and embedding protocols used by the demo.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Function struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}
type Call struct {
	Function Function `json:"function"`
}
type Message struct {
	Role     string `json:"role"`
	Content  string `json:"content"`
	ToolName string `json:"tool_name,omitempty"`
	Calls    []Call `json:"tool_calls,omitempty"`
}
type Tool struct {
	Type     string     `json:"type"`
	Function Definition `json:"function"`
}
type Definition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}
type Request struct {
	Model    string         `json:"model"`
	Messages []Message      `json:"messages"`
	Tools    []Tool         `json:"tools,omitempty"`
	Stream   bool           `json:"stream"`
	Think    bool           `json:"think"`
	Options  map[string]int `json:"options"`
}
type Response struct {
	Message      Message `json:"message"`
	Done         bool    `json:"done"`
	Error        string  `json:"error"`
	PromptTokens int     `json:"prompt_eval_count"`
}
type Client struct {
	Base string
	HTTP *http.Client
}

func New(base string) *Client {
	return &Client{
		strings.TrimRight(base, "/"),
		&http.Client{
			Timeout:       5 * time.Minute,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (c *Client) Chat(
	ctx context.Context,
	messages []Message,
	tools []Tool,
	emit func(string) error,
) (Response, error) {
	stream := emit != nil

	var last Response

	b, err := json.Marshal(
		Request{
			Model:    "qwen3.5:4b",
			Messages: messages,
			Tools:    tools,
			Stream:   stream,
			Think:    false,
			Options:  map[string]int{"num_ctx": 4096, "num_predict": 512},
		},
	)
	if err != nil {
		return last, err
	}

	req, err := c.request(ctx, "POST", "/api/chat", b)
	if err != nil {
		return last, err
	}

	res, err := c.HTTP.Do(req)
	if err != nil {
		return last, errors.New("ollama unavailable")
	}

	defer func() {
		if err := res.Body.Close(); err != nil {
			slog.Warn("resource cleanup failed", "error", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return last, errors.New("ollama rejected request")
	}

	return decodeChat(io.LimitReader(res.Body, 2<<20), emit)
}

func decodeChat(reader io.Reader, emit func(string) error) (Response, error) {
	var last Response

	stream := emit != nil
	d := json.NewDecoder(reader)

	for {
		var r Response
		if err := d.Decode(&r); err != nil {
			if errors.Is(err, io.EOF) {
				err = errors.New("incomplete Ollama response")
			}

			return last, err
		}

		if r.Error != "" {
			return last, errors.New("ollama generation failed")
		}

		last = r
		if len(r.Message.Calls) > 0 && stream {
			return last, errors.New("unexpected final tool call")
		}

		if stream && r.Message.Content != "" {
			if err := emit(r.Message.Content); err != nil {
				return last, err
			}
		}

		if !stream || r.Done {
			return last, nil
		}
	}
}

func (c *Client) request(
	ctx context.Context,
	method, path string,
	data []byte,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, bytes.NewReader(data))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, err
}
