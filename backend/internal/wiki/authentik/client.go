package authentik

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ai-buddy/internal/wiki"
)

type Client struct {
	URL, Token string
	Client     *http.Client
}

func New(url, token string) *Client {
	return &Client{
		URL:   strings.TrimRight(url, "/"),
		Token: token,
		Client: &http.Client{
			Timeout:       10 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (a *Client) Team(ctx context.Context, id string) (string, error) {
	pk, err := strconv.ParseInt(id, 10, 64)
	if err != nil || pk <= 0 {
		return "", wiki.ErrNotFound
	}
	req, err := http.NewRequestWithContext(ctx, "GET", a.URL+"/api/v3/core/users/"+strconv.FormatInt(pk, 10)+"/", nil)
	if err != nil {
		return "", wiki.ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+a.Token)

	resp, err := a.Client.Do(req)
	if err != nil {
		return "", wiki.ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return "", wiki.ErrNotFound
	}
	if resp.StatusCode != 200 {
		return "", wiki.ErrUnavailable
	}

	var user struct {
		PK         int64 `json:"pk"`
		Active     bool  `json:"is_active"`
		Attributes struct {
			Team string `json:"team"`
		} `json:"attributes"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&user) != nil || user.PK != pk {
		return "", wiki.ErrUnavailable
	}
	if !user.Active || strings.TrimSpace(user.Attributes.Team) == "" {
		return "", wiki.ErrNotFound
	}
	return user.Attributes.Team, nil
}
