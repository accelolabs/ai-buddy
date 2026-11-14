// Package remote provides bounded, authenticated requests to Buddy dependencies.
package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	Base, Key string
	HTTP      *http.Client
}

func ValidURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}
func New(base, key string, timeout time.Duration) *Client {
	return &Client{strings.TrimRight(base, "/"), key, &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (c *Client) Do(ctx context.Context, method, path, key string, input, output any) error {
	var body []byte
	var err error
	if input != nil {
		body, err = json.Marshal(input)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Key)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("dependency unavailable: %w", err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			slog.Warn("resource cleanup failed", "error", err)
		}
	}()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("dependency returned HTTP %d", res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return fmt.Errorf("invalid dependency response")
	}
	if json.Unmarshal(data, output) != nil {
		return fmt.Errorf("invalid dependency JSON")
	}
	return nil
}
