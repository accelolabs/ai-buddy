package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"

	"ai-buddy/backend-ticket/internal/apperror"
	"ai-buddy/backend-ticket/internal/tickets"
)

type Repository struct {
	db *sql.DB
}

type scanner interface {
	Scan(...any) error
}

const columns = `
	id,
	sender_id,
	recipient_id,
	question,
	description,
	created_at
`

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func scan(row scanner) (tickets.Ticket, error) {
	var ticket tickets.Ticket
	err := row.Scan(
		&ticket.ID,
		&ticket.SenderID,
		&ticket.RecipientID,
		&ticket.Question,
		&ticket.Description,
		&ticket.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = apperror.ErrNotFound
	}

	return ticket, err
}

func (p *Repository) Find(ctx context.Context, sender, key string) (tickets.Ticket, error) {
	const query = `
  SELECT ` + columns + `
  FROM tickets
  WHERE sender_id=$1 AND idempotency_key=$2
 `

	return scan(p.db.QueryRowContext(
		ctx,
		query,
		sender,
		key,
	))
}

func (p *Repository) Create(ctx context.Context, sender, recipient, key string, input tickets.CreateInput) (tickets.Ticket, bool, error) {
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return tickets.Ticket{}, false, err
	}
	ticket, err := scan(p.db.QueryRowContext(
		ctx,
		`INSERT INTO tickets (
   id,
   sender_id,
   recipient_id,
   question,
   description,
   idempotency_key
  )
  VALUES ($1,$2,$3,$4,$5,$6)
  ON CONFLICT (sender_id,idempotency_key) DO NOTHING
  RETURNING `+columns,
		hex.EncodeToString(token),
		sender,
		recipient,
		input.Question,
		input.Description,
		key,
	))
	if err == nil {
		return ticket, true, nil
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		return tickets.Ticket{}, false, err
	}

	// The INSERT waits for the winner; this SELECT uses a fresh committed snapshot.
	ticket, err = p.Find(ctx, sender, key)
	if err != nil {
		return tickets.Ticket{}, false, err
	}
	if ticket.Question != input.Question || ticket.Description != input.Description {
		return tickets.Ticket{}, false, apperror.ErrConflict
	}

	return ticket, false, nil
}

func (p *Repository) list(ctx context.Context, query, id string) ([]tickets.Ticket, error) {
	rows, err := p.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []tickets.Ticket{}
	for rows.Next() {
		ticket, err := scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, ticket)
	}

	return list, rows.Err()
}

func (p *Repository) Sent(ctx context.Context, id string) ([]tickets.Ticket, error) {
	const query = `
  SELECT ` + columns + `
  FROM tickets
  WHERE sender_id=$1
  ORDER BY created_at,id
 `

	return p.list(ctx, query, id)
}

func (p *Repository) Incoming(ctx context.Context, id string) ([]tickets.Ticket, error) {
	const query = `
  SELECT ` + columns + `
  FROM tickets
  WHERE recipient_id=$1
  ORDER BY created_at,id
 `

	return p.list(ctx, query, id)
}
