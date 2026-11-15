package repository

import (
	"backend/internal/buddy"
	"context"
	"fmt"
)

func (r *Repository) AddMessage(ctx context.Context, message buddy.Message) (buddy.Message, error) {
	err := r.db.QueryRowContext(ctx, `
  INSERT INTO messages (chat_id, role, content) VALUES ($1, $2, $3)
  RETURNING id, created_at`, message.ChatID, message.Role, message.Content).
		Scan(&message.ID, &message.CreatedAt)
	if err != nil {
		return buddy.Message{}, fmt.Errorf("add message: %w", err)
	}
	return message, nil
}

func (r *Repository) ListMessages(ctx context.Context, chatID string) ([]buddy.Message, error) {
	rows, err := r.db.QueryContext(ctx, `
  SELECT id, chat_id, role, content, created_at FROM messages WHERE chat_id = $1 ORDER BY id`, chatID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer func() { _ = rows.Close() }()
	messages := make([]buddy.Message, 0)
	for rows.Next() {
		var message buddy.Message
		if err := rows.Scan(&message.ID, &message.ChatID, &message.Role, &message.Content, &message.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read messages: %w", err)
	}
	return messages, nil
}
