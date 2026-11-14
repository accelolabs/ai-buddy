package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"ai-buddy/internal/wiki"
)

type Client struct {
	URL    string
	Client *http.Client
}

func New(url string) *Client {
	return &Client{
		URL: strings.TrimRight(url, "/"),
		Client: &http.Client{
			Timeout:       60 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (o *Client) request(ctx context.Context, method, path string, body any, out any) error {
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			return wiki.ErrUnavailable
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, o.URL+path, bytes.NewReader(data))
	if err != nil {
		return wiki.ErrUnavailable
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.Client.Do(req)
	if err != nil {
		return wiki.ErrUnavailable
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("resource cleanup failed", "error", err)
		}
	}()

	if resp.StatusCode != 200 { // Ollama reports context overflow as a 400.
		var failure struct {
			Error string `json:"error"`
		}
		if json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&failure) != nil {
			return wiki.ErrUnavailable
		}
		if resp.StatusCode == 400 && (strings.Contains(failure.Error, "context length") || strings.Contains(failure.Error, "input length")) {
			return wiki.ErrContext
		}
		return wiki.ErrUnavailable
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out) != nil {
		return wiki.ErrUnavailable
	}
	return nil
}

func (o *Client) Fingerprint(ctx context.Context) (string, error) {
	var body struct {
		Models []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"models"`
	}
	if err := o.request(ctx, "GET", "/api/tags", nil, &body); err != nil {
		return "", err
	}
	for _, m := range body.Models {
		if m.Name == wiki.Model && m.Digest != "" {
			return wiki.Model + ":" + m.Digest + ":" + wiki.SplitVersion, nil
		}
	}
	return "", wiki.ErrUnavailable
}

func (o *Client) Embed(ctx context.Context, input string) ([]float64, error) {
	var body struct {
		Embeddings [][]float64 `json:"embeddings"`
	}
	err := o.request(ctx, "POST", "/api/embed", struct {
		Model    string `json:"model"`
		Input    string `json:"input"`
		Truncate bool   `json:"truncate"`
	}{Model: wiki.Model, Input: input, Truncate: false}, &body)
	if err != nil {
		return nil, err
	}
	if len(body.Embeddings) != 1 || len(body.Embeddings[0]) != 768 {
		return nil, wiki.ErrUnavailable
	}

	var norm float64
	for _, value := range body.Embeddings[0] {
		norm += value * value
	}
	if norm == 0 {
		return nil, wiki.ErrUnavailable
	}
	return body.Embeddings[0], nil
}
