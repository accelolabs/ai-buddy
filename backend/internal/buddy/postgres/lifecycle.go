package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"ai-buddy/internal/buddy"
)

func (p *Repository) Finish(ctx context.Context, id, failure string) error {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Warn("resource cleanup failed", "error", err)
		}
	}()
	var status string
	err = tx.QueryRowContext(ctx, "SELECT status FROM runs WHERE id=$1 FOR UPDATE", id).Scan(&status)
	if err != nil {
		return missing(err)
	}
	if status == buddy.RunDone || status == buddy.RunError {
		return nil
	}
	status = buddy.RunDone
	kind := buddy.EventDone
	if failure != "" {
		status = buddy.RunError
		kind = buddy.EventError
	}
	_, err = tx.ExecContext(ctx, "UPDATE runs SET status=$1,error=$2 WHERE id=$3", status, failure, id)
	if err != nil {
		return err
	}
	if err = appendEvent(ctx, tx, id, kind, map[string]string{"status": status, "error": failure}); err != nil {
		return err
	}
	return tx.Commit()
}
func (p *Repository) Recover(ctx context.Context) error {
	rows, err := p.DB.QueryContext(ctx, "SELECT id FROM runs WHERE status IN ('queued','running')")
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			defer func() {
				if err := rows.Close(); err != nil {
					slog.Warn("resource cleanup failed", "error", err)
				}
			}()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	if err := rows.Close(); err != nil {
		slog.Warn("resource cleanup failed", "error", err)
	}
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err = p.Finish(ctx, id, "service restarted; retry manually"); err != nil {
			return err
		}
	}
	_, err = p.DB.ExecContext(ctx, "UPDATE actions SET status='error',error='service restarted; retry manually' WHERE status='executing'")
	return err
}
