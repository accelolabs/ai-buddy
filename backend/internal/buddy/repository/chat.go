package repository

import (
	"backend/internal/buddy"
	"context"
	"database/sql"
	"fmt"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) AddChat(ctx context.Context, chat buddy.Chat) (buddy.Chat, error) {
	err := r.db.QueryRowContext(ctx, `
  INSERT INTO chats (id, user_id, title) VALUES ($1, $2, $3)
  RETURNING created_at`, chat.ID, chat.UserID, chat.Title).Scan(&chat.CreatedAt)
	if err != nil {
		return buddy.Chat{}, fmt.Errorf("add chat: %w", err)
	}
	return chat, nil
}

func (r *Repository) GetChat(ctx context.Context, id string) (buddy.Chat, error) {
	var chat buddy.Chat
	err := r.db.QueryRowContext(ctx, `SELECT id, user_id, title, created_at FROM chats WHERE id = $1`, id).
		Scan(&chat.ID, &chat.UserID, &chat.Title, &chat.CreatedAt)
	if err != nil {
		return buddy.Chat{}, fmt.Errorf("get chat: %w", err)
	}
	return chat, nil
}

func (r *Repository) ListChats(ctx context.Context, userID string) ([]buddy.Chat, error) {
	rows, err := r.db.QueryContext(ctx, `
  SELECT id, user_id, title, created_at FROM chats WHERE user_id = $1
  ORDER BY created_at DESC, id DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	defer func() { _ = rows.Close() }()
	chats := make([]buddy.Chat, 0)
	for rows.Next() {
		var chat buddy.Chat
		if err := rows.Scan(&chat.ID, &chat.UserID, &chat.Title, &chat.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan chat: %w", err)
		}
		chats = append(chats, chat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read chats: %w", err)
	}
	return chats, nil
}
