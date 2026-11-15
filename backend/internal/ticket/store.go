package ticket

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"

	"ai-buddy/internal/platform/database"
)

type Store struct {
	DB *sql.DB
}

const columns = `
	id,
	sender_id,
	recipient_id,
	question,
	description,
	created_at
`

func scan(row database.RowScanner) (Ticket, error) {
	var ticket Ticket
	err := row.Scan(
		&ticket.ID,
		&ticket.SenderID,
		&ticket.RecipientID,
		&ticket.Question,
		&ticket.Description,
		&ticket.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}

	return ticket, err
}

func (p *Store) Find(ctx context.Context, sender, key string) (Ticket, error) {
	const query = `
  SELECT ` + columns + `
  FROM tickets
  WHERE sender_id=$1 AND idempotency_key=$2
 `

	return scan(p.DB.QueryRowContext(
		ctx,
		query,
		sender,
		key,
	))
}

func (p *Store) Create(ctx context.Context, sender, recipient, key string, input CreateInput) (Ticket, bool, error) {
	var token [16]byte
	_, _ = rand.Read(token[:]) // crypto/rand.Read always fills the buffer or terminates.
	ticket, err := scan(p.DB.QueryRowContext(
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
		hex.EncodeToString(token[:]),
		sender,
		recipient,
		input.Question,
		input.Description,
		key,
	))
	if err == nil {
		return ticket, true, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Ticket{}, false, err
	}

	// The INSERT waits for the winner; this SELECT uses a fresh committed snapshot.
	ticket, err = p.Find(ctx, sender, key)
	if err != nil {
		return Ticket{}, false, err
	}
	if ticket.Question != input.Question || ticket.Description != input.Description {
		return Ticket{}, false, ErrConflict
	}

	return ticket, false, nil
}

func (p *Store) list(ctx context.Context, query, id string) ([]Ticket, error) {
	return database.Query(ctx, p.DB, query, scan, id)
}

func (p *Store) Sent(ctx context.Context, id string) ([]Ticket, error) {
	const query = `
  SELECT ` + columns + `
  FROM tickets
  WHERE sender_id=$1
  ORDER BY created_at,id
 `

	return p.list(ctx, query, id)
}

func (p *Store) Incoming(ctx context.Context, id string) ([]Ticket, error) {
	const query = `
  SELECT ` + columns + `
  FROM tickets
  WHERE recipient_id=$1
  ORDER BY created_at,id
 `

	return p.list(ctx, query, id)
}
