package repository

import (
	"backend/internal/buddy"
	"context"
	"fmt"
)

func (r *Repository) AddMessage(ctx context.Context, message buddy.Message) (buddy.Message, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return buddy.Message{}, fmt.Errorf("begin message: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	err = tx.QueryRowContext(ctx, `
  INSERT INTO messages (chat_id, role, content, attachment) VALUES ($1, $2, $3, $4)
  RETURNING id, created_at`, message.ChatID, message.Role, message.Content, message.Source != nil).
		Scan(&message.ID, &message.CreatedAt)
	if err != nil {
		return buddy.Message{}, fmt.Errorf("add message: %w", err)
	}
	if source := message.Source; source != nil {
		_, err := tx.ExecContext(ctx, `
   INSERT INTO message_sources (message_id, document_id, ordinal, title, anchor, quote)
   VALUES ($1, $2, $3, $4, $5, $6)`, message.ID, source.DocumentID, source.Ordinal, source.Title, source.Anchor, source.Quote)
		if err != nil {
			return buddy.Message{}, fmt.Errorf("add message source: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return buddy.Message{}, fmt.Errorf("commit message: %w", err)
	}
	return message, nil
}

func (r *Repository) ListMessages(ctx context.Context, chatID string) ([]buddy.Message, error) {
	rows, err := r.db.QueryContext(ctx, `
  SELECT id, chat_id, role, content, created_at, attachment FROM messages WHERE chat_id = $1 ORDER BY id`, chatID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer func() { _ = rows.Close() }()
	messages := make([]buddy.Message, 0)
	attachments := make(map[int64]int)
	for rows.Next() {
		var message buddy.Message
		var attachment bool
		if err := rows.Scan(&message.ID, &message.ChatID, &message.Role, &message.Content, &message.CreatedAt, &attachment); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		if attachment {
			attachments[message.ID] = len(messages)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read messages: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close messages: %w", err)
	}
	if len(attachments) == 0 {
		return messages, nil
	}

	sources, err := r.db.QueryContext(ctx, `
  SELECT s.message_id, s.document_id, s.ordinal, s.title, s.anchor, s.quote
  FROM message_sources AS s JOIN messages AS m ON m.id = s.message_id
  WHERE m.chat_id = $1 AND m.attachment`, chatID)
	if err != nil {
		return nil, fmt.Errorf("list message sources: %w", err)
	}
	defer func() { _ = sources.Close() }()
	for sources.Next() {
		var messageID int64
		var source buddy.Source
		if err := sources.Scan(&messageID, &source.DocumentID, &source.Ordinal, &source.Title, &source.Anchor, &source.Quote); err != nil {
			return nil, fmt.Errorf("scan message source: %w", err)
		}
		if index, ok := attachments[messageID]; ok {
			messages[index].Source = &source
		}
	}
	if err := sources.Err(); err != nil {
		return nil, fmt.Errorf("read message sources: %w", err)
	}
	return messages, nil
}

// ListRecentMessages returns the last limit messages in chronological order, without sources.
func (r *Repository) ListRecentMessages(ctx context.Context, chatID string, limit int) ([]buddy.Message, error) {
	rows, err := r.db.QueryContext(ctx, `
  SELECT id, chat_id, role, content, created_at FROM (
    SELECT id, chat_id, role, content, created_at FROM messages
    WHERE chat_id = $1 ORDER BY id DESC LIMIT $2
  ) AS recent ORDER BY id`, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent messages: %w", err)
	}
	defer func() { _ = rows.Close() }()
	messages := make([]buddy.Message, 0)
	for rows.Next() {
		var message buddy.Message
		if err := rows.Scan(&message.ID, &message.ChatID, &message.Role, &message.Content, &message.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recent message: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read recent messages: %w", err)
	}
	return messages, nil
}
