package tracker

import (
	"backend-ai/internal/platform/database"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
)

type Store struct {
	DB *sql.DB
}

const taskColumns = `
	id,
	intern_id,
	title,
	description,
	to_char(deadline,'YYYY-MM-DD'),
	created_at,
	completed_at
`

func scanTask(row database.RowScanner) (Task, error) {
	var task Task

	err := row.Scan(
		&task.ID,
		&task.InternID,
		&task.Title,
		&task.Description,
		&task.Deadline,
		&task.CreatedAt,
		&task.CompletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}

	return task, err
}

func (p *Store) Tasks(ctx context.Context, id string) ([]Task, error) {
	const query = `
		SELECT ` + taskColumns + `
		FROM tasks
		WHERE intern_id=$1
		ORDER BY created_at,id
	`

	return database.Query(ctx, p.DB, query, scanTask, id)
}

func (p *Store) Create(ctx context.Context, id, key string, input CreateInput) (Task, bool, error) {
	const insertQuery = `
		INSERT INTO tasks(
			id,intern_id,team,title,description,deadline,idempotency_key
		)
		SELECT $1,id,team,$3,$4,$5,$6
		FROM profiles
		WHERE id=$2 AND role='intern'
		ON CONFLICT (intern_id,idempotency_key) DO NOTHING
		RETURNING ` + taskColumns

	var token [16]byte

	_, _ = rand.Read(token[:]) // crypto/rand.Read always fills the buffer or terminates.

	// A conflicting INSERT waits for the winning transaction. The following SELECT
	// gets a fresh READ COMMITTED snapshot and sees its committed task.
	row := p.DB.QueryRowContext(
		ctx,
		insertQuery,
		hex.EncodeToString(token[:]),
		id,
		input.Title,
		input.Description,
		input.Deadline,
		key,
	)

	task, err := scanTask(row)
	if err == nil {
		return task, true, nil
	}

	if !errors.Is(err, ErrNotFound) {
		return Task{}, false, err
	}

	const findQuery = `
		SELECT ` + taskColumns + `
		FROM tasks
		WHERE intern_id=$1 AND idempotency_key=$2
	`

	row = p.DB.QueryRowContext(ctx, findQuery, id, key)

	task, err = scanTask(row)
	if err != nil {
		return Task{}, false, err
	}

	if task.Title != input.Title ||
		task.Description != input.Description ||
		task.Deadline != input.Deadline {
		return Task{}, false, ErrConflict
	}

	return task, false, nil
}

func (p *Store) Complete(ctx context.Context, id, taskID string) (Task, error) {
	const query = `
		UPDATE tasks
		SET completed_at=COALESCE(completed_at,now())
		WHERE intern_id=$1 AND id=$2
		RETURNING ` + taskColumns

	row := p.DB.QueryRowContext(ctx, query, id, taskID)

	return scanTask(row)
}
