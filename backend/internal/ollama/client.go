package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	contextLength int
	baseURL       string
	model         string
	http          *http.Client
}

func NewClient(baseURL, model string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		http:    &http.Client{Timeout: time.Minute},
	}
}

// NewChatClient configures the context window for dialogue requests.
func NewChatClient(baseURL, model string, contextLength int) *Client {
	client := NewClient(baseURL, model)
	client.contextLength = contextLength
	return client
}

type embedRequest struct {
	Model    string `json:"model"`
	Input    string `json:"input"`
	Truncate bool   `json:"truncate"`
}

type embedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
	Error      string      `json:"error"`
}

func (c *Client) Embed(ctx context.Context, input string) ([]float64, error) {
	body, err := json.Marshal(embedRequest{Model: c.model, Input: input, Truncate: false})
	if err != nil {
		return nil, fmt.Errorf("encode embedding request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request embedding: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embedding: HTTP %d", response.StatusCode)
	}
	var result embedResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embedding: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("ollama embedding: %s", result.Error)
	}
	if len(result.Embeddings) != 1 || len(result.Embeddings[0]) == 0 {
		return nil, fmt.Errorf("ollama embedding: expected one nonempty vector")
	}
	return result.Embeddings[0], nil
}
