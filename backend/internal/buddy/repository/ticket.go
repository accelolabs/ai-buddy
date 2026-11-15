package repository

import (
	"backend/internal/buddy"
	"backend/internal/ticket"
	ticketrepository "backend/internal/ticket/repository"
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
)

// ConfirmTicket locks the owned card, so retries and concurrent confirmations reuse its result.
func (r *Repository) ConfirmTicket(ctx context.Context, userID, chatID string, messageID int64, title, description string) (buddy.Ticket, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return buddy.Ticket{}, false, fmt.Errorf("begin ticket confirmation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var card buddy.Ticket
	err = tx.QueryRowContext(ctx, `SELECT c.title, c.description, c.recipient_id, c.recipient_name, c.ticket_id
 FROM message_tickets c JOIN messages m ON m.id = c.message_id JOIN chats ch ON ch.id = m.chat_id
 WHERE c.message_id = $1 AND m.chat_id = $2 AND ch.user_id = $3 AND m.role = 'assistant'
 FOR UPDATE OF c`, messageID, chatID, userID).
		Scan(&card.Title, &card.Description, &card.Recipient.ID, &card.Recipient.Name, &card.TicketID)
	if err != nil {
		return buddy.Ticket{}, false, fmt.Errorf("lock ticket card: %w", err)
	}
	created := card.TicketID == nil
	if created {
		// Lock the assignment too, so a concurrent reassignment cannot invalidate this check before commit.
		var mentorID string
		err := tx.QueryRowContext(ctx, `SELECT mentor_id FROM mentorships WHERE intern_id = $1 FOR SHARE`, userID).Scan(&mentorID)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && mentorID != card.Recipient.ID) {
			return buddy.Ticket{}, false, buddy.ErrMentorChanged
		}
		if err != nil {
			return buddy.Ticket{}, false, fmt.Errorf("check assigned mentor: %w", err)
		}
		value, err := ticketrepository.NewRepository(tx).Add(ctx, ticket.Ticket{ID: rand.Text(), UserID: userID, Title: title, Description: description})
		if err != nil {
			return buddy.Ticket{}, false, err
		}
		_, err = tx.ExecContext(ctx, `UPDATE message_tickets SET title = $2, description = $3, ticket_id = $4 WHERE message_id = $1`, messageID, title, description, value.ID)
		if err != nil {
			return buddy.Ticket{}, false, fmt.Errorf("link created ticket: %w", err)
		}
		card.Title, card.Description, card.TicketID = title, description, &value.ID
	}
	if err := tx.Commit(); err != nil {
		return buddy.Ticket{}, false, fmt.Errorf("commit ticket confirmation: %w", err)
	}
	return card, created, nil
}
