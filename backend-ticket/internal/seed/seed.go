package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"ai-buddy/backend-ticket/internal/tickets"
	"ai-buddy/backend-ticket/internal/tracker"
)

type Config struct {
	URL, Token               string
	TrackerURL, TicketAPIKey string
	TrackerTimeout           time.Duration
}

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

type user struct {
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
		URL:            strings.TrimRight(os.Getenv("AUTHENTIK_URL"), "/"),
		Token:          os.Getenv("AUTHENTIK_API_TOKEN"),
		TrackerURL:     os.Getenv("TRACKER_URL"),
		TicketAPIKey:   os.Getenv("TICKET_API_KEY"),
		TrackerTimeout: 5 * time.Second,
	}

	if raw := os.Getenv("TRACKER_TIMEOUT"); raw != "" {
		var err error
		c.TrackerTimeout, err = time.ParseDuration(raw)
		if err != nil {
			return Config{}, errors.New("invalid tracker timeout")
		}
	}
	u, err := url.Parse(c.URL)
	if !tracker.ValidURL(c.TrackerURL) || strings.TrimSpace(c.TicketAPIKey) == "" || c.TrackerTimeout <= 0 || err != nil ||
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

func lookup(ctx context.Context, client *http.Client, c Config, d demo) (user, error) {
	endpoint := c.URL + "/api/v3/core/users/?username=" + url.QueryEscape(d.Username)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return user{}, errors.New("invalid Authentik URL")
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := client.Do(req)
	if err != nil {
		return user{}, errors.New("Authentik request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return user{}, errors.New("Authentik request rejected")
	}

	var body struct {
		Results    []user `json:"results"`
		Pagination struct {
			Next int `json:"next"`
		} `json:"pagination"`
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 || json.Unmarshal(raw, &body) != nil || body.Pagination.Next != 0 {
		return user{}, errors.New("invalid Authentik response")
	}

	var found user
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
		return user{}, fmt.Errorf("invalid demo identity: %s", d.Username)
	}

	return found, nil
}

func Run(ctx context.Context, db *sql.DB, c Config) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return RunWithClient(ctx, db, c, client, time.Now())
}

// Complete external reads before starting the transaction.
func RunWithClient(ctx context.Context, db *sql.DB, c Config, client *http.Client, now time.Time) error {
	finder := tracker.New(c.TrackerURL, c.TicketAPIKey, c.TrackerTimeout)
	return run(ctx, db, c, client, finder, now)
}

type mentorFinder interface {
	Mentor(context.Context, string) (string, error)
}

func run(ctx context.Context, db *sql.DB, c Config, client *http.Client, finder mentorFinder, now time.Time) error {
	users := make([]user, 0, len(demos))
	ids := map[int64]bool{}
	for _, d := range demos {
		u, err := lookup(ctx, client, c, d)
		if err != nil {
			return err
		}
		if ids[u.PK] {
			return errors.New("duplicate demo identity")
		}
		ids[u.PK] = true
		users = append(users, u)
	}

	list := []tickets.Ticket{}
	for _, u := range users {
		if u.Attributes.Role != "intern" {
			continue
		}
		id := strconv.FormatInt(u.PK, 10)
		recipient, err := finder.Mentor(ctx, id)
		if err != nil {
			return err
		}
		if recipient == "" {
			continue
		}
		list = append(list, tickets.Ticket{
			ID:          "demo:ticket:" + u.Username,
			SenderID:    id,
			RecipientID: recipient,
			Question:    "Как лучше выполнить первую задачу?",
			Description: "Нужна помощь с архитектурой проекта",
			CreatedAt:   now,
		})
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, ticket := range list {
		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO tickets (
    id,
    sender_id,
    recipient_id,
    question,
    description,
    created_at,
    idempotency_key
   )
   VALUES ($1,$2,$3,$4,$5,$6,$7)
   ON CONFLICT DO NOTHING`,
			ticket.ID,
			ticket.SenderID,
			ticket.RecipientID,
			ticket.Question,
			ticket.Description,
			ticket.CreatedAt,
			ticket.ID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
