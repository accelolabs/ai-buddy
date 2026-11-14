package tracker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ai-buddy/backend-ticket/internal/apperror"
)

type Client struct {
	baseURL string
	key     string
	http    *http.Client
}

func ValidURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}

func New(baseURL, key string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		key:     key,
		http: &http.Client{
			Timeout:       timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (c *Client) Mentor(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/internal/interns/"+url.PathEscape(id)+"/mentor", nil)
	if err != nil {
		return "", apperror.ErrTracker
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", apperror.ErrTracker
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return "", apperror.ErrNotFound
	}
	if resp.StatusCode != 200 {
		return "", apperror.ErrTracker
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return "", apperror.ErrTracker
	}
	var result map[string]json.RawMessage
	if json.Unmarshal(body, &result) != nil {
		return "", apperror.ErrTracker
	}
	raw, ok := result["mentor"]
	if !ok {
		return "", apperror.ErrTracker
	}
	if string(raw) == "null" {
		return "", nil
	}
	var mentor struct {
		ID string `json:"authentik_id"`
	}
	if json.Unmarshal(raw, &mentor) != nil || strings.TrimSpace(mentor.ID) == "" || strings.TrimSpace(mentor.ID) != mentor.ID {
		return "", apperror.ErrTracker
	}
	return mentor.ID, nil
}
