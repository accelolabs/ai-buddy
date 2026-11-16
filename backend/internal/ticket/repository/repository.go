package repository

import (
	"context"
	"database/sql"
	"fmt"

	// Project packages.
	"backend/internal/ticket"
	"backend/internal/users"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Add(ctx context.Context, value ticket.Ticket) (ticket.Ticket, error) {
	err := r.db.QueryRowContext(
		ctx,
		`
			INSERT INTO tickets (id, user_id, title, description)
			VALUES ($1, $2, $3, $4)
			RETURNING created_at
		`,
		value.ID,
		value.UserID,
		value.Title,
		value.Description,
	).Scan(&value.CreatedAt)
	if err != nil {
		return ticket.Ticket{}, fmt.Errorf("add ticket: %w", err)
	}

	return value, nil
}

func (r *Repository) List(ctx context.Context, user users.User) ([]ticket.Ticket, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
			SELECT t.id, t.user_id, t.title, t.description, t.created_at
			FROM tickets t
			WHERE (
				($1 = 'intern' AND t.user_id = $2)
				OR (
					$1 = 'mentor'
					AND EXISTS (
						SELECT 1
						FROM mentorships m
						WHERE m.intern_id = t.user_id
							AND m.mentor_id = $2
					)
				)
			)
			ORDER BY t.created_at DESC, t.id DESC
		`,
		user.Role,
		user.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	values := make([]ticket.Ticket, 0)

	for rows.Next() {
		var value ticket.Ticket
		if err := rows.Scan(&value.ID, &value.UserID, &value.Title, &value.Description, &value.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan ticket: %w", err)
		}

		values = append(values, value)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tickets: %w", err)
	}

	return values, nil
}

func (r *Repository) Get(ctx context.Context, user users.User, id string) (ticket.Ticket, error) {
	var value ticket.Ticket
	err := r.db.QueryRowContext(
		ctx,
		`
			SELECT t.id, t.user_id, t.title, t.description, t.created_at
			FROM tickets t
			WHERE (
				($1 = 'intern' AND t.user_id = $2)
				OR (
					$1 = 'mentor'
					AND EXISTS (
						SELECT 1
						FROM mentorships m
						WHERE m.intern_id = t.user_id
							AND m.mentor_id = $2
					)
				)
			)
				AND t.id = $3
		`,
		user.Role,
		user.ID,
		id,
	).
		Scan(&value.ID, &value.UserID, &value.Title, &value.Description, &value.CreatedAt)
	if err != nil {
		return ticket.Ticket{}, fmt.Errorf("get ticket: %w", err)
	}

	return value, nil
}
