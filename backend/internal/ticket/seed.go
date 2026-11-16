package ticket

import (
	"context"
	"database/sql"
	"fmt"
)

// Seed adds demo tickets without replacing existing records.
func Seed(ctx context.Context, db *sql.DB) error {
	records := []struct {
		title, description string
	}{
		{title: "Доступ к репозиторию", description: "Прошу предоставить доступ к репозиторию команды для выполнения задач стажировки."},
		{title: "Локальный запуск", description: "Подскажите порядок локального запуска приложения и необходимые настройки."},
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin ticket seed: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, internID := range []string{"intern.backend.1", "intern.backend.2", "intern.platform.1", "intern.platform.2"} {
		for i, record := range records {
			id := fmt.Sprintf("demo.ticket.%s.%d", internID, i+1)
			_, err := tx.ExecContext(ctx, `INSERT INTO tickets (id, user_id, title, description) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO NOTHING`,
				id, internID, record.title, record.description)
			if err != nil {
				return fmt.Errorf("seed %q: %w", id, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit ticket seed: %w", err)
	}
	return nil
}
