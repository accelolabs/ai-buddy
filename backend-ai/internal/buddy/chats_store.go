package buddy

import (
	"backend-ai/internal/platform/cleanup"
	"backend-ai/internal/platform/database"
	"context"
	"database/sql"
)

func (p *Store) CreateChat(ctx context.Context, owner string) (c Chat, err error) {
	c.ID = ID()
	c.Owner = owner
	err = p.DB.QueryRowContext(ctx, "INSERT INTO chats(id,owner_id) VALUES($1,$2) RETURNING created_at", c.ID, owner).
		Scan(&c.CreatedAt)

	return c, err
}

func scanChat(row database.RowScanner) (c Chat, err error) {
	err = row.Scan(&c.ID, &c.Owner, &c.Title, &c.CreatedAt)
	return c, missing(err)
}

func scanMessage(row database.RowScanner) (m Message, err error) {
	err = row.Scan(&m.ID, &m.ChatID, &m.RunID, &m.Role, &m.Content, &m.CreatedAt)
	return m, err
}

func scanCard(row database.RowScanner) (c Card, err error) {
	err = row.Scan(&c.RunID, &c.Kind, &c.Data)
	return c, err
}

func (p *Store) Chats(ctx context.Context, owner string) ([]Chat, error) {
	return database.Query(
		ctx,
		p.DB,
		"SELECT id,owner_id,title,created_at FROM chats WHERE owner_id=$1 ORDER BY created_at DESC,id",
		scanChat,
		owner,
	)
}

func (p *Store) History(ctx context.Context, owner, id string) (h History, err error) {
	tx, err := p.DB.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true},
	)
	if err != nil {
		return h, err
	}
	defer cleanup.Rollback(tx)

	h.Chat, err = scanChat(
		tx.QueryRowContext(
			ctx,
			"SELECT id,owner_id,title,created_at FROM chats WHERE id=$1 AND owner_id=$2",
			id,
			owner,
		),
	)
	if err != nil {
		return h, err
	}

	h.Messages, err = database.Query(
		ctx,
		tx,
		"SELECT id,chat_id,run_id,role,content,created_at FROM messages WHERE chat_id=$1 ORDER BY created_at,role DESC,id",
		scanMessage,
		id,
	)
	if err != nil {
		return h, err
	}

	h.Runs, err = database.Query(
		ctx,
		tx,
		"SELECT "+runCols+" FROM runs WHERE chat_id=$1 ORDER BY created_at,id",
		scanRun,
		id,
	)
	if err != nil {
		return h, err
	}

	h.Actions, err = database.Query(
		ctx,
		tx,
		"SELECT "+actionCols+" FROM actions WHERE run_id IN (SELECT id FROM runs WHERE chat_id=$1) ORDER BY id",
		scanAction,
		id,
	)
	if err != nil {
		return h, err
	}

	h.Cards, err = database.Query(
		ctx,
		tx,
		"SELECT e.run_id,e.kind,e.data FROM events e JOIN runs r ON r.id=e.run_id WHERE r.chat_id=$1 AND e.kind IN ('source','result') ORDER BY r.created_at,e.sequence",
		scanCard,
		id,
	)
	if err != nil {
		return h, err
	}

	return h, tx.Commit()
}
