// Package seed creates demo profiles and sample data transactionally.
package seed

import (
	"backend-ai/internal/platform/cleanup"
	"backend-ai/internal/users"
	"context"
	"database/sql"
	"fmt"
	"time"
)

func Run(ctx context.Context, db *sql.DB) error {
	now := time.Now()

	const (
		insertProfile = `
		INSERT INTO profiles(
			id,name,username,email,role,team,start_date,goals,expectations
		)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT DO NOTHING
	`
		insertAssignment = `
		INSERT INTO assignments(intern_id,mentor_id,team)
		VALUES($1,$2,$3)
		ON CONFLICT (intern_id) DO NOTHING
	`
		insertTask = `
		INSERT INTO tasks(
			id,intern_id,team,title,description,deadline,
			created_at,completed_at,idempotency_key
		)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT DO NOTHING
	`
	)

	demos := []struct{ Username, Name, Role, Team string }{
		{"mentor.backend", "Алексей Смирнов", users.RoleMentor, backendTeam},
		{"intern.backend.1", "Иван Петров", users.RoleIntern, backendTeam},
		{"intern.backend.2", "Анна Соколова", users.RoleIntern, backendTeam},
		{"mentor.platform", "Мария Волкова", users.RoleMentor, platformTeam},
		{"intern.platform.1", "Дмитрий Орлов", users.RoleIntern, platformTeam},
		{"intern.platform.2", "Елена Морозова", users.RoleIntern, platformTeam},
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer cleanup.Rollback(tx)

	mentors := map[string]string{}

	for _, u := range demos {
		id := u.Username

		var start any

		goals, expectations := "", ""

		if u.Role == users.RoleIntern {
			start = now.AddDate(0, 0, -7).Format(time.DateOnly)
			goals = "Освоить проект и выполнить первую задачу"
			expectations = "Регулярная обратная связь и поддержка наставника"
		} else {
			mentors[u.Team] = id
		}

		_, err = tx.ExecContext(
			ctx,
			insertProfile,
			id,
			u.Name,
			u.Username,
			u.Username+"@example.test",
			u.Role,
			u.Team,
			start,
			goals,
			expectations,
		)
		if err != nil {
			return err
		}
	}

	for _, u := range demos {
		if u.Role != users.RoleIntern {
			continue
		}

		id := u.Username
		if u.Username != "intern.platform.2" {
			_, err = tx.ExecContext(
				ctx,
				insertAssignment,
				id,
				mentors[u.Team],
				u.Team,
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
				u.Team,
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

		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO tickets(id,sender_id,recipient_id,question,description,idempotency_key)
   SELECT $1,a.intern_id,a.mentor_id,$2,$3,$1 FROM assignments a WHERE a.intern_id=$4 ON CONFLICT DO NOTHING`,
			"demo:ticket:"+u.Username,
			"Как лучше выполнить первую задачу?",
			"Нужна помощь с архитектурой проекта",
			u.Username,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

const (
	backendTeam  = "backend"
	platformTeam = "platform"
)
