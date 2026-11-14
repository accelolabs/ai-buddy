package demousers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct{ URL, Token string }
type demo struct {
	Username, Role, Team string
}

var demos = []demo{
	{"mentor.backend", "mentor", "backend"},
	{"intern.backend.1", "intern", "backend"},
	{"intern.backend.2", "intern", "backend"},
	{"mentor.platform", "mentor", "platform"},
	{"intern.platform.1", "intern", "platform"},
	{"intern.platform.2", "intern", "platform"},
}

type User struct {
	PK         int64  `json:"pk"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Active     bool   `json:"is_active"`
	Attributes struct {
		Role string `json:"role"`
		Team string `json:"team"`
	} `json:"attributes"`
}

func LoadConfig() (Config, error) {
	c := Config{
		URL:   strings.TrimRight(os.Getenv("AUTHENTIK_URL"), "/"),
		Token: os.Getenv("AUTHENTIK_API_TOKEN"),
	}

	u, err := url.Parse(c.URL)
	if err != nil ||
		(u.Scheme != "https" && u.Scheme != "http") ||
		u.Hostname() == "" ||
		u.User != nil ||
		u.RawQuery != "" ||
		u.Fragment != "" ||
		strings.TrimSpace(c.Token) == "" {
		return Config{}, errors.New("invalid seed Authentik configuration")
	}

	return c, nil
}

func lookup(ctx context.Context, client *http.Client, c Config, d demo) (User, error) {
	endpoint := c.URL + "/api/v3/core/users/?username=" + url.QueryEscape(d.Username)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return User{}, errors.New("invalid Authentik URL")
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := client.Do(req)
	if err != nil {
		return User{}, errors.New("Authentik request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return User{}, errors.New("Authentik request rejected")
	}

	var body struct {
		Results    []User `json:"results"`
		Pagination struct {
			Next int `json:"next"`
		} `json:"pagination"`
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 || json.Unmarshal(raw, &body) != nil || body.Pagination.Next != 0 {
		return User{}, errors.New("invalid Authentik response")
	}

	var found User
	count := 0
	for _, u := range body.Results {
		if u.Username == d.Username {
			found = u
			count++
		}
	}

	if count != 1 ||
		found.PK <= 0 ||
		!found.Active ||
		found.Attributes.Role != d.Role ||
		found.Attributes.Team != d.Team {
		return User{}, fmt.Errorf("invalid demo identity: %s", d.Username)
	}

	return found, nil
}

func Client() *http.Client {
	return &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}
func Load(ctx context.Context, client *http.Client, c Config) ([]User, error) {
	users := make([]User, 0, len(demos))
	ids := map[int64]bool{}
	for _, d := range demos {
		u, err := lookup(ctx, client, c, d)
		if err != nil {
			return nil, err
		}

		if ids[u.PK] {
			return nil, errors.New("duplicate demo identity")
		}

		ids[u.PK] = true
		users = append(users, u)
	}

	return users, nil
}
