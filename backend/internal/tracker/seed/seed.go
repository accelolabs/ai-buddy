package seed

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ai-buddy/internal/platform/demousers"
)

type Config = demousers.Config

var LoadConfig = demousers.LoadConfig

func Run(ctx context.Context, db *sql.DB, c Config) error {
	client := demousers.Client()

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

	users, err := demousers.Load(ctx, client, demousers.Config{URL: c.URL, Token: c.Token})
	if err != nil {
		return err
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
