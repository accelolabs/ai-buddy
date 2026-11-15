package buddy

import (
	"ai-buddy/internal/platform/cleanup"
	"context"
	"errors"
)

func (p *Store) Start(ctx context.Context, owner, chat, key, text string) (Run, bool, error) {
	return p.startRun(ctx, owner, chat, "message:"+key, text, "")
}

func (p *Store) Retry(ctx context.Context, owner, id string) (Run, bool, error) {
	r, err := p.Run(ctx, owner, id)
	if err != nil {
		return r, false, err
	}

	if r.Status != RunError {
		return r, false, ErrConflict
	}

	return p.startRun(ctx, owner, r.ChatID, "retry:"+id, "", id)
}

func (p *Store) startRun(
	ctx context.Context,
	owner, chat, key, text, retry string,
) (r Run, fresh bool, err error) {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return r, false, err
	}
	defer cleanup.Rollback(tx)

	var title string

	err = tx.QueryRowContext(ctx, "SELECT title FROM chats WHERE id=$1 AND owner_id=$2 FOR UPDATE", chat, owner).
		Scan(&title)
	if err != nil {
		return r, false, missing(err)
	}

	r, err = scanRun(
		tx.QueryRowContext(
			ctx,
			"SELECT "+runCols+" FROM runs WHERE chat_id=$1 AND request_key=$2",
			chat,
			key,
		),
	)
	if err == nil {
		var old string

		err = tx.QueryRowContext(ctx, "SELECT content FROM messages WHERE id=$1", r.MessageID).
			Scan(&old)
		if err != nil {
			return r, false, err
		}

		if retry == "" && old != text {
			return r, false, ErrConflict
		}

		return r, false, nil
	}

	if !errors.Is(err, ErrNotFound) {
		return r, false, err
	}

	var active bool

	err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM runs WHERE chat_id=$1 AND status IN ('queued','running'))", chat).
		Scan(&active)
	if err != nil {
		return r, false, err
	}

	if active {
		return r, false, ErrConflict
	}

	r = Run{ID: ID(), ChatID: chat, Owner: owner, Status: RunQueued}

	var retryID any

	if retry != "" {
		retryID = retry
		err = tx.QueryRowContext(ctx, "SELECT message_id FROM runs WHERE id=$1 AND status='error'", retry).
			Scan(&r.MessageID)
	} else {
		r.MessageID = ID()
		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO messages(id,chat_id,run_id,role,content) VALUES($1,$2,$3,'user',$4)",
			r.MessageID,
			chat,
			r.ID,
			text,
		)
	}

	if err != nil {
		return r, false, err
	}

	if title == "" {
		_, err = tx.ExecContext(
			ctx,
			"UPDATE chats SET title=$1 WHERE id=$2",
			Title(text),
			chat,
		)
		if err != nil {
			return r, false, err
		}
	}

	err = tx.QueryRowContext(ctx, "INSERT INTO runs(id,chat_id,owner_id,message_id,status,request_key,retry_of) VALUES($1,$2,$3,$4,'queued',$5,$6) RETURNING created_at", r.ID, chat, owner, r.MessageID, key, retryID).
		Scan(&r.CreatedAt)
	if err != nil {
		return r, false, err
	}

	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO messages(id,chat_id,run_id,role,content) VALUES($1,$2,$3,'assistant','')",
		ID(),
		chat,
		r.ID,
	)
	if err != nil {
		return r, false, err
	}

	if err := tx.Commit(); err != nil {
		return r, false, err
	}

	return r, true, nil
}

func (p *Store) Run(ctx context.Context, owner, id string) (Run, error) {
	return scanRun(
		p.DB.QueryRowContext(
			ctx,
			"SELECT "+runCols+" FROM runs WHERE id=$1 AND owner_id=$2",
			id,
			owner,
		),
	)
}
