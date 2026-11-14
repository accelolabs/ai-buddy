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
)

type Config struct {
	URL, Token string
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
	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if dec.Decode(&body) != nil || body.Pagination.Next != 0 {
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

// Fetch every identity before opening the transaction: API failures cannot leave
// partially populated profiles. Inserts never overwrite live application state.
func RunWithClient(ctx context.Context, db *sql.DB, c Config, client *http.Client, now time.Time) error {
	const insertProfile = `
		INSERT INTO profiles(
			authentik_id,name,username,email,role,team,start_date,goals,expectations
		)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT DO NOTHING
	`
	const insertAssignment = `
		INSERT INTO assignments(intern_id,mentor_id,team)
		VALUES($1,$2,$3)
		ON CONFLICT (intern_id) DO NOTHING
	`
	const insertTask = `
		INSERT INTO tasks(
			id,intern_id,team,title,description,deadline,
			created_at,completed_at,idempotency_key
		)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT DO NOTHING
	`

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

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	mentors := map[string]string{}
	for _, u := range users {
		id := strconv.FormatInt(u.PK, 10)
		var start any
		goals, expectations := "", ""
		if u.Attributes.Role == "intern" {
			start = now.AddDate(0, 0, -7).Format(time.DateOnly)
			goals = "Освоить проект и выполнить первую задачу"
			expectations = "Регулярная обратная связь и поддержка наставника"
		} else {
			mentors[u.Attributes.Team] = id
		}

		_, err = tx.ExecContext(
			ctx,
			insertProfile,
			id,
			u.Name,
			u.Username,
			u.Email,
			u.Attributes.Role,
			u.Attributes.Team,
			start,
			goals,
			expectations,
		)
		if err != nil {
			return err
		}
	}

	for _, u := range users {
		if u.Attributes.Role != "intern" {
			continue
		}

		id := strconv.FormatInt(u.PK, 10)
		if u.Username != "intern.platform.2" {
			_, err = tx.ExecContext(
				ctx,
				insertAssignment,
				id,
				mentors[u.Attributes.Team],
				u.Attributes.Team,
			)
			if err != nil {
				return err
			}
		}

		deadlineOffsets := []int{-3, -1, 7}
		for i, title := range []string{
			"Познакомиться с командой",
			"Изучить архитектуру",
			"Выполнить первую задачу",
		} {
			var completed any
			if i == 0 {
				completed = now.AddDate(0, 0, -3)
			}

			createdAt := now.AddDate(0, 0, -7).Add(time.Duration(i) * time.Second)
			deadline := now.AddDate(0, 0, deadlineOffsets[i]).Format(time.DateOnly)
			key := fmt.Sprintf("demo:%s:%d", u.Username, i)
			_, err = tx.ExecContext(
				ctx,
				insertTask,
				key,
				id,
				u.Attributes.Team,
				title,
				"Демонстрационная задача",
				deadline,
				createdAt,
				completed,
				key,
			)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
