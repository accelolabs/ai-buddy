package buddy

import (
	"backend-ai/internal/platform/cleanup"
	"backend-ai/internal/platform/database"
	"context"
)

func (p *Store) Finish(ctx context.Context, id, failure string) error {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer cleanup.Rollback(tx)

	var status string

	err = tx.QueryRowContext(ctx, "SELECT status FROM runs WHERE id=$1 FOR UPDATE", id).
		Scan(&status)
	if err != nil {
		return missing(err)
	}

	if status == RunDone || status == RunError {
		return nil
	}

	status = RunDone
	kind := EventDone

	if failure != "" {
		status = RunError
		kind = EventError
	}

	_, err = tx.ExecContext(
		ctx,
		"UPDATE runs SET status=$1,error=$2 WHERE id=$3",
		status,
		failure,
		id,
	)
	if err != nil {
		return err
	}

	if err := appendEvent(
		ctx,
		tx,
		id,
		kind,
		map[string]string{"status": status, "error": failure},
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (p *Store) Recover(ctx context.Context) error {
	ids, err := database.Query(
		ctx,
		p.DB,
		"SELECT id FROM runs WHERE status IN ('queued','running')",
		func(row database.RowScanner) (id string, err error) { err = row.Scan(&id); return id, err },
	)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if err := p.Finish(ctx, id, "service restarted; retry manually"); err != nil {
			return err
		}
	}

	_, err = p.DB.ExecContext(
		ctx,
		"UPDATE actions SET status='error',error='service restarted; retry manually' WHERE status='executing'",
	)

	return err
}
