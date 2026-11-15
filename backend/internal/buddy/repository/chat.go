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
