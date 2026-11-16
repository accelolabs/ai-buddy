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
		hours       int
		completed   bool
	}{
		{
			description: "Изучить документацию",
			hours:       72,
		},
		{
			description: "Настроить окружение",
			hours:       -24,
		},
		{
			description: "Ознакомиться с командой",
			hours:       -48,
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
					VALUES ($1, $2, $3, now() + $4 * interval '1 hour', $5)
					ON CONFLICT (id) DO NOTHING
				`,
				id,
				internID,
				record.description,
				record.hours,
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
