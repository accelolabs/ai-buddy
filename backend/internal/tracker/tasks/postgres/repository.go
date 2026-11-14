package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"

	"ai-buddy/internal/tracker/apperror"
	"ai-buddy/internal/tracker/tasks"
)

type Repository struct {
	db *sql.DB
}

type scanner interface {
	Scan(...any) error
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

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func scanTask(row scanner) (tasks.Task, error) {
	var task tasks.Task
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
		err = apperror.ErrNotFound
	}

	return task, err
}

func (p *Repository) Tasks(ctx context.Context, id string) ([]tasks.Task, error) {
	const query = `
		SELECT ` + taskColumns + `
		FROM tasks
		WHERE intern_id=$1
		ORDER BY created_at,id
	`

	rows, err := p.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []tasks.Task{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		list = append(list, task)
	}

	return list, rows.Err()
}

func (p *Repository) Create(ctx context.Context, id, key string, input tasks.CreateInput) (tasks.Task, bool, error) {
	const insertQuery = `
		INSERT INTO tasks(
			id,intern_id,team,title,description,deadline,idempotency_key
		)
		SELECT $1,authentik_id,team,$3,$4,$5,$6
		FROM profiles
		WHERE authentik_id=$2 AND role='intern'
		ON CONFLICT (intern_id,idempotency_key) DO NOTHING
		RETURNING ` + taskColumns

	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return tasks.Task{}, false, err
	}

	// A conflicting INSERT waits for the winning transaction. The following SELECT
	// gets a fresh READ COMMITTED snapshot and sees its committed task.
	row := p.db.QueryRowContext(
		ctx,
		insertQuery,
		hex.EncodeToString(token),
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
	if !errors.Is(err, apperror.ErrNotFound) {
		return tasks.Task{}, false, err
	}

	const findQuery = `
		SELECT ` + taskColumns + `
		FROM tasks
		WHERE intern_id=$1 AND idempotency_key=$2
	`

	row = p.db.QueryRowContext(ctx, findQuery, id, key)
	task, err = scanTask(row)
	if err != nil {
		return tasks.Task{}, false, err
	}

	if task.Title != input.Title ||
		task.Description != input.Description ||
		task.Deadline != input.Deadline {
		return tasks.Task{}, false, apperror.ErrConflict
	}

	return task, false, nil
}

func (p *Repository) Complete(ctx context.Context, id, taskID string) (tasks.Task, error) {
	const query = `
		UPDATE tasks
		SET completed_at=COALESCE(completed_at,now())
		WHERE intern_id=$1 AND id=$2
		RETURNING ` + taskColumns

	row := p.db.QueryRowContext(ctx, query, id, taskID)
	return scanTask(row)
}
