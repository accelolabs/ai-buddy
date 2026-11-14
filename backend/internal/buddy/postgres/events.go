package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/platform/cleanup"
)

func appendEvent(ctx context.Context, tx *sql.Tx, id, kind string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO events(run_id,sequence,kind,data) SELECT $1,COALESCE(MAX(sequence),0)+1,$2,$3 FROM events WHERE run_id=$1", id, kind, b)
	return err
}
func (p *Repository) Event(ctx context.Context, id, kind string, data any) error {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer cleanup.Rollback(tx)
	var status string
	err = tx.QueryRowContext(ctx, "SELECT status FROM runs WHERE id=$1 FOR UPDATE", id).Scan(&status)
	if err != nil {
		return missing(err)
	}
	if status != buddy.RunQueued && status != buddy.RunRunning {
		return buddy.ErrConflict
	}
	switch kind {
	case buddy.EventText:
		text, ok := data.(string)
		if !ok {
			return buddy.ErrInvalid
		}
		_, err = tx.ExecContext(ctx, "UPDATE messages SET content=content || $1 WHERE run_id=$2 AND role='assistant'", text, id)
	case buddy.EventStage:
		_, err = tx.ExecContext(ctx, "UPDATE runs SET status='running' WHERE id=$1", id)
	}
	if err != nil {
		return err
	}
	if err = appendEvent(ctx, tx, id, kind, data); err != nil {
		return err
	}
	return tx.Commit()
}
func (p *Repository) Events(ctx context.Context, owner, id string, after int64) ([]buddy.Event, buddy.Run, error) {
	r, err := p.Run(ctx, owner, id)
	if err != nil {
		return nil, r, err
	}
	rows, err := p.DB.QueryContext(ctx, "SELECT sequence,kind,data FROM events WHERE run_id=$1 AND sequence>$2 ORDER BY sequence", id, after)
	if err != nil {
		return nil, r, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Warn("resource cleanup failed", "error", err)
		}
	}()
	out := []buddy.Event{}
	for rows.Next() {
		var e buddy.Event
		if err = rows.Scan(&e.ID, &e.Kind, &e.Data); err != nil {
			return nil, r, err
		}
		out = append(out, e)
	}
	return out, r, rows.Err()
}
