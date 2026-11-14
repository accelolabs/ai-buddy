package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"ai-buddy/internal/buddy"
)

func (p *Repository) CreateChat(ctx context.Context, owner string) (c buddy.Chat, err error) {
	c.ID = buddy.ID()
	c.Owner = owner
	err = p.DB.QueryRowContext(ctx, "INSERT INTO chats(id,owner_id) VALUES($1,$2) RETURNING created_at", c.ID, owner).Scan(&c.CreatedAt)
	return
}
func (p *Repository) Chats(ctx context.Context, owner string) ([]buddy.Chat, error) {
	rows, err := p.DB.QueryContext(ctx, "SELECT id,owner_id,title,created_at FROM chats WHERE owner_id=$1 ORDER BY created_at DESC,id", owner)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Warn("resource cleanup failed", "error", err)
		}
	}()
	out := []buddy.Chat{}
	for rows.Next() {
		var c buddy.Chat
		if err = rows.Scan(&c.ID, &c.Owner, &c.Title, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (p *Repository) History(ctx context.Context, owner, id string) (h buddy.History, err error) {
	tx, err := p.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return h, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Warn("resource cleanup failed", "error", err)
		}
	}()
	err = tx.QueryRowContext(ctx, "SELECT id,owner_id,title,created_at FROM chats WHERE id=$1 AND owner_id=$2", id, owner).Scan(&h.Chat.ID, &h.Chat.Owner, &h.Chat.Title, &h.Chat.CreatedAt)
	if err != nil {
		return h, missing(err)
	}
	h.Messages = []buddy.Message{}
	h.Runs = []buddy.Run{}
	h.Actions = []buddy.Action{}
	rows, err := tx.QueryContext(ctx, "SELECT id,chat_id,run_id,role,content,created_at FROM messages WHERE chat_id=$1 ORDER BY created_at,role DESC,id", id)
	if err != nil {
		return h, err
	}
	for rows.Next() {
		var m buddy.Message
		if err = rows.Scan(&m.ID, &m.ChatID, &m.RunID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			defer func() {
				if err := rows.Close(); err != nil {
					slog.Warn("resource cleanup failed", "error", err)
				}
			}()
			return h, err
		}
		h.Messages = append(h.Messages, m)
	}
	err = rows.Err()
	if err := rows.Close(); err != nil {
		slog.Warn("resource cleanup failed", "error", err)
	}
	if err != nil {
		return h, err
	}
	rows, err = tx.QueryContext(ctx, "SELECT "+runCols+" FROM runs WHERE chat_id=$1 ORDER BY created_at,id", id)
	if err != nil {
		return h, err
	}
	for rows.Next() {
		r, e := scanRun(rows)
		if e != nil {
			defer func() {
				if err := rows.Close(); err != nil {
					slog.Warn("resource cleanup failed", "error", err)
				}
			}()
			return h, e
		}
		h.Runs = append(h.Runs, r)
	}
	err = rows.Err()
	if err := rows.Close(); err != nil {
		slog.Warn("resource cleanup failed", "error", err)
	}
	if err != nil {
		return h, err
	}
	rows, err = tx.QueryContext(ctx, "SELECT "+actionCols+" FROM actions WHERE run_id IN (SELECT id FROM runs WHERE chat_id=$1) ORDER BY id", id)
	if err != nil {
		return h, err
	}
	for rows.Next() {
		a, e := scanAction(rows)
		if e != nil {
			defer func() {
				if err := rows.Close(); err != nil {
					slog.Warn("resource cleanup failed", "error", err)
				}
			}()
			return h, e
		}
		h.Actions = append(h.Actions, a)
	}
	err = rows.Err()
	if err := rows.Close(); err != nil {
		slog.Warn("resource cleanup failed", "error", err)
	}
	if err != nil {
		return h, err
	}

	h.Cards = []buddy.Card{}
	rows, err = tx.QueryContext(ctx, "SELECT e.run_id,e.kind,e.data FROM events e JOIN runs r ON r.id=e.run_id WHERE r.chat_id=$1 AND e.kind IN ('source','result') ORDER BY r.created_at,e.sequence", id)
	if err != nil {
		return h, err
	}
	for rows.Next() {
		var c buddy.Card
		if err = rows.Scan(&c.RunID, &c.Kind, &c.Data); err != nil {
			defer func() {
				if err := rows.Close(); err != nil {
					slog.Warn("resource cleanup failed", "error", err)
				}
			}()
			return h, err
		}
		h.Cards = append(h.Cards, c)
	}
	err = rows.Err()
	if err := rows.Close(); err != nil {
		slog.Warn("resource cleanup failed", "error", err)
	}
	if err != nil {
		return h, err
	}
	err = tx.Commit()
	return
}
