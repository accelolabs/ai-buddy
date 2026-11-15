package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Message struct {
	Role     string `json:"role"`
	Content  string `json:"content"`
	ToolName string `json:"tool_name,omitempty"`
	Calls    []Call `json:"tool_calls,omitempty"`
}

type Call struct {
	Function Function `json:"function"`
}

type Function struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
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

type chatRequest struct {
	Model    string      `json:"model"`
	Messages []Message   `json:"messages"`
	Tools    []Tool      `json:"tools,omitempty"`
	Stream   bool        `json:"stream"`
	Think    bool        `json:"think"`
	Options  chatOptions `json:"options"`
}

type chatOptions struct {
	Context int `json:"num_ctx"`
	Predict int `json:"num_predict"`
}

type Response struct {
	Message Message `json:"message"`
	Done    bool    `json:"done"`
	Error   string  `json:"error"`
}

func (c *Client) Chat(ctx context.Context, messages []Message, tools []Tool, emit func(string) error) (Response, error) {
	body, err := json.Marshal(chatRequest{
		Model: c.model, Messages: messages, Tools: tools, Stream: emit != nil,
		Think: false, Options: chatOptions{Context: c.contextLength, Predict: 512},
	})
	if err != nil {
		return Response{}, fmt.Errorf("encode chat request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("request chat: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("ollama chat: HTTP %d", response.StatusCode)
	}
	decoder := json.NewDecoder(response.Body)
	var text strings.Builder
	var calls []Call
	for {
		var result Response
		if err := decoder.Decode(&result); err != nil {
			return Response{}, fmt.Errorf("decode chat: %w", err)
		}
		if result.Error != "" {
			return Response{}, fmt.Errorf("ollama chat: %s", result.Error)
		}
		text.WriteString(result.Message.Content)
		calls = append(calls, result.Message.Calls...)
		if emit != nil && result.Message.Content != "" {
			if err := emit(result.Message.Content); err != nil {
				return Response{}, err
			}
		}
		if result.Done {
			result.Message.Content = text.String()
			result.Message.Calls = calls
			return result, nil
		}
		if emit == nil {
			return Response{}, fmt.Errorf("ollama chat: incomplete response")
		}
	}
}
