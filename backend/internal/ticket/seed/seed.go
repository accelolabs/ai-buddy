package seed

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"ai-buddy/internal/platform/cleanup"
	"ai-buddy/internal/platform/demousers"
	"ai-buddy/internal/ticket/tickets"
	"ai-buddy/internal/ticket/tracker"
)

type Config struct {
	URL, Token               string
	TrackerURL, TicketAPIKey string
	TrackerTimeout           time.Duration
}

func LoadConfig() (Config, error) {
	authConfig, err := demousers.LoadConfig()
	if err != nil {
		return Config{}, err
	}
	c := Config{
		URL:            authConfig.URL,
		Token:          authConfig.Token,
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
	if !tracker.ValidURL(c.TrackerURL) || strings.TrimSpace(c.TicketAPIKey) == "" || c.TrackerTimeout <= 0 {
		return Config{}, errors.New("invalid tracker configuration")
	}

	return c, nil
}

func Run(ctx context.Context, db *sql.DB, c Config) error {
	client := demousers.Client()

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
	users, err := demousers.Load(ctx, client, demousers.Config{URL: c.URL, Token: c.Token})
	if err != nil {
		return err
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
	defer cleanup.Rollback(tx)

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
