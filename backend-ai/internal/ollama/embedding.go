package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const EmbeddingModel = "embeddinggemma:300m"

var (
	ErrUnavailable = errors.New("search dependency unavailable")
	ErrContext     = errors.New("embedding context exceeded")
)

func (c *Client) decodeRequest(ctx context.Context, method, path string, body, out any) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var data []byte

	if body != nil {
		var err error

		data, err = json.Marshal(body)
		if err != nil {
			return ErrUnavailable
		}
	}

	req, err := c.request(ctx, method, path, data)
	if err != nil {
		return ErrUnavailable
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return ErrUnavailable
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("resource cleanup failed", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK { // Ollama reports context overflow as a 400.
		var failure struct {
			Error string `json:"error"`
		}
		if json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&failure) != nil {
			return ErrUnavailable
		}

		if resp.StatusCode == http.StatusBadRequest &&
			(strings.Contains(failure.Error, "context length") || strings.Contains(failure.Error, "input length")) {
			return ErrContext
		}

		return ErrUnavailable
	}

	if json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out) != nil {
		return ErrUnavailable
	}

	return nil
}

func (c *Client) Fingerprint(ctx context.Context) (string, error) {
	var body struct {
		Models []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"models"`
	}
	if err := c.decodeRequest(ctx, "GET", "/api/tags", nil, &body); err != nil {
		return "", err
	}

	for _, m := range body.Models {
		if m.Name == EmbeddingModel && m.Digest != "" {
			return EmbeddingModel + ":" + m.Digest, nil
		}
	}

	return "", ErrUnavailable
}

func (c *Client) Embed(ctx context.Context, input string) ([]float64, error) {
	var body struct {
		Embeddings [][]float64 `json:"embeddings"`
	}

	err := c.decodeRequest(ctx, "POST", "/api/embed", struct {
		Model    string `json:"model"`
		Input    string `json:"input"`
		Truncate bool   `json:"truncate"`
	}{Model: EmbeddingModel, Input: input, Truncate: false}, &body)
	if err != nil {
		return nil, err
	}

	if len(body.Embeddings) != 1 || len(body.Embeddings[0]) != 768 {
		return nil, ErrUnavailable
	}

	var norm float64

	for _, value := range body.Embeddings[0] {
		norm += value * value
	}

	if norm == 0 {
		return nil, ErrUnavailable
	}

	return body.Embeddings[0], nil
}
