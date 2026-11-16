package tracker

import (
	"context"
	"database/sql"
	"fmt"
)

// Seed adds demo tasks without replacing existing records.
func Seed(ctx context.Context, db *sql.DB) error {
	records := []struct {
		description string
		days        int
		completed   bool
	}{
		{
			description: "Изучить документацию",
			days:        3,
		},
		{
			description: "Настроить окружение",
			days:        -1,
		},
		{
			description: "Ознакомиться с командой",
			days:        -2,
			completed:   true,
		},
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tracker seed: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	for _, internID := range []string{"intern.backend.1", "intern.backend.2", "intern.platform.1", "intern.platform.2"} {
		for i, record := range records {
			id := fmt.Sprintf("demo.task.%s.%d", internID, i+1)
			_, err := tx.ExecContext(
				ctx,
				`
					INSERT INTO tasks (id, user_id, description, deadline, completed)
					VALUES ($1, $2, $3, (now() AT TIME ZONE 'Europe/Moscow')::date + $4::integer, $5)
					ON CONFLICT (id) DO NOTHING
				`,
				id,
				internID,
				record.description,
				record.days,
				record.completed,
			)
			if err != nil {
				return fmt.Errorf("seed %q: %w", id, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tracker seed: %w", err)
	}

	return nil
}
