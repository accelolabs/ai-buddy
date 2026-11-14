package tracker

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
)

type Postgres struct{ DB *sql.DB }
type scanner interface{ Scan(...any) error }

const profileColumns = `authentik_id,name,username,email,role,team,to_char(start_date,'YYYY-MM-DD'),goals,expectations`

func scanProfile(row scanner) (Profile, error) {
	var p Profile
	e := row.Scan(&p.ID, &p.Name, &p.Username, &p.Email, &p.Role, &p.Team, &p.StartDate, &p.Goals, &p.Expectations)
	if errors.Is(e, sql.ErrNoRows) {
		e = ErrNotFound
	}
	return p, e
}
func (p *Postgres) Intern(ctx context.Context, id string) (Profile, error) {
	return scanProfile(p.DB.QueryRowContext(ctx, `SELECT `+profileColumns+` FROM profiles WHERE authentik_id=$1 AND role='intern'`, id))
}
func (p *Postgres) Mentor(ctx context.Context, id string) (*Profile, error) {
	v, e := scanProfile(p.DB.QueryRowContext(ctx, `SELECT `+profileColumns+` FROM profiles WHERE authentik_id=(SELECT mentor_id FROM assignments WHERE intern_id=$1)`, id))
	if errors.Is(e, ErrNotFound) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return &v, nil
}
func (p *Postgres) Interns(ctx context.Context, id string) ([]Profile, error) {
	rows, e := p.DB.QueryContext(ctx, `SELECT `+profileColumns+` FROM profiles WHERE authentik_id IN (SELECT intern_id FROM assignments WHERE mentor_id=$1) ORDER BY authentik_id`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Profile{}
	for rows.Next() {
		v, e := scanProfile(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (p *Postgres) Assigned(ctx context.Context, mentor, id string) (bool, error) {
	var ok bool
	e := p.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM assignments WHERE mentor_id=$1 AND intern_id=$2)`, mentor, id).Scan(&ok)
	return ok, e
}

const taskColumns = `id,intern_id,title,description,to_char(deadline,'YYYY-MM-DD'),created_at,completed_at`

func scanTask(row scanner) (Task, error) {
	var t Task
	e := row.Scan(&t.ID, &t.InternID, &t.Title, &t.Description, &t.Deadline, &t.CreatedAt, &t.CompletedAt)
	if errors.Is(e, sql.ErrNoRows) {
		e = ErrNotFound
	}
	return t, e
}
func (p *Postgres) Tasks(ctx context.Context, id string) ([]Task, error) {
	rows, e := p.DB.QueryContext(ctx, `SELECT `+taskColumns+` FROM tasks WHERE intern_id=$1 ORDER BY created_at,id`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Task{}
	for rows.Next() {
		v, e := scanTask(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (p *Postgres) Create(ctx context.Context, id, key string, in Input) (Task, bool, error) {
	// A conflicting INSERT waits for the winning transaction. The following SELECT
	// gets a fresh READ COMMITTED snapshot and sees its committed task.
	token := make([]byte, 16)
	if _, e := rand.Read(token); e != nil {
		return Task{}, false, e
	}
	t, e := scanTask(p.DB.QueryRowContext(ctx, `INSERT INTO tasks(id,intern_id,team,title,description,deadline,idempotency_key)
 SELECT $1,authentik_id,team,$3,$4,$5,$6 FROM profiles WHERE authentik_id=$2 AND role='intern'
 ON CONFLICT (intern_id,idempotency_key) DO NOTHING RETURNING `+taskColumns, hex.EncodeToString(token), id, in.Title, in.Description, in.Deadline, key))
	if e == nil {
		return t, true, nil
	}
	if !errors.Is(e, ErrNotFound) {
		return Task{}, false, e
	}
	t, e = scanTask(p.DB.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks WHERE intern_id=$1 AND idempotency_key=$2`, id, key))
	if e != nil {
		return Task{}, false, e
	}
	if t.Title != in.Title || t.Description != in.Description || t.Deadline != in.Deadline {
		return Task{}, false, ErrConflict
	}
	return t, false, nil
}
func (p *Postgres) Complete(ctx context.Context, id, taskID string) (Task, error) {
	return scanTask(p.DB.QueryRowContext(ctx, `UPDATE tasks SET completed_at=COALESCE(completed_at,now()) WHERE intern_id=$1 AND id=$2 RETURNING `+taskColumns, id, taskID))
}
