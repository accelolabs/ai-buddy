package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"

	"ai-buddy/backend-tracker/internal/apperror"
	"ai-buddy/backend-tracker/internal/tasks"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

type scanner interface{ Scan(...any) error }

const taskColumns = `id,intern_id,title,description,to_char(deadline,'YYYY-MM-DD'),created_at,completed_at`

func scanTask(row scanner) (tasks.Task, error) {
	var t tasks.Task
	err := row.Scan(&t.ID, &t.InternID, &t.Title, &t.Description, &t.Deadline, &t.CreatedAt, &t.CompletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		err = apperror.ErrNotFound
	}
	return t, err
}
func (p *Repository) Tasks(ctx context.Context, id string) ([]tasks.Task, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT `+taskColumns+` FROM tasks WHERE intern_id=$1 ORDER BY created_at,id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []tasks.Task{}
	for rows.Next() {
		v, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (p *Repository) Create(ctx context.Context, id, key string, in tasks.CreateInput) (tasks.Task, bool, error) {
	// A conflicting INSERT waits for the winning transaction. The following SELECT
	// gets a fresh READ COMMITTED snapshot and sees its committed task.
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return tasks.Task{}, false, err
	}
	t, err := scanTask(p.db.QueryRowContext(ctx, `INSERT INTO tasks(id,intern_id,team,title,description,deadline,idempotency_key)
 SELECT $1,authentik_id,team,$3,$4,$5,$6 FROM profiles WHERE authentik_id=$2 AND role='intern'
 ON CONFLICT (intern_id,idempotency_key) DO NOTHING RETURNING `+taskColumns, hex.EncodeToString(token), id, in.Title, in.Description, in.Deadline, key))
	if err == nil {
		return t, true, nil
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		return tasks.Task{}, false, err
	}
	t, err = scanTask(p.db.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks WHERE intern_id=$1 AND idempotency_key=$2`, id, key))
	if err != nil {
		return tasks.Task{}, false, err
	}
	if t.Title != in.Title || t.Description != in.Description || t.Deadline != in.Deadline {
		return tasks.Task{}, false, apperror.ErrConflict
	}
	return t, false, nil
}
func (p *Repository) Complete(ctx context.Context, id, taskID string) (tasks.Task, error) {
	return scanTask(p.db.QueryRowContext(ctx, `UPDATE tasks SET completed_at=COALESCE(completed_at,now()) WHERE intern_id=$1 AND id=$2 RETURNING `+taskColumns, id, taskID))
}
