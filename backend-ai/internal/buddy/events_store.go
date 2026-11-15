package buddy

import (
	"backend-ai/internal/platform/cleanup"
	"backend-ai/internal/platform/database"
	"context"
	"database/sql"
	"encoding/json"
)

func appendEvent(ctx context.Context, tx *sql.Tx, id, kind string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO events(run_id,sequence,kind,data) SELECT $1,COALESCE(MAX(sequence),0)+1,$2,$3 FROM events WHERE run_id=$1",
		id,
		kind,
		b,
	)

	return err
}

func (p *Store) Event(ctx context.Context, id, kind string, data any) error {
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

	if status != RunQueued && status != RunRunning {
		return ErrConflict
	}

	switch kind {
	case EventText:
		text, ok := data.(string)
		if !ok {
			return ErrInvalid
		}

		_, err = tx.ExecContext(
			ctx,
			"UPDATE messages SET content=content || $1 WHERE run_id=$2 AND role='assistant'",
			text,
			id,
		)
	case EventStage:
		_, err = tx.ExecContext(ctx, "UPDATE runs SET status='running' WHERE id=$1", id)
	}

	if err != nil {
		return err
	}

	if err := appendEvent(ctx, tx, id, kind, data); err != nil {
		return err
	}

	return tx.Commit()
}

func (p *Store) Events(ctx context.Context, owner, id string, after int64) ([]Event, Run, error) {
	r, err := p.Run(ctx, owner, id)
	if err != nil {
		return nil, r, err
	}

	out, err := database.Query(
		ctx,
		p.DB,
		"SELECT sequence,kind,data FROM events WHERE run_id=$1 AND sequence>$2 ORDER BY sequence",
		scanEvent,
		id,
		after,
	)

	return out, r, err
}

func scanEvent(row database.RowScanner) (e Event, err error) {
	err = row.Scan(&e.ID, &e.Kind, &e.Data)
	return e, err
}
