package postgres

import (
	"context"
	"database/sql"
	"errors"

	"ai-buddy/backend-tracker/internal/apperror"
	"ai-buddy/backend-tracker/internal/interns"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

type scanner interface{ Scan(...any) error }

const profileColumns = `authentik_id,name,username,email,role,team,to_char(start_date,'YYYY-MM-DD'),goals,expectations`

func scanProfile(row scanner) (interns.Profile, error) {
	var p interns.Profile
	err := row.Scan(&p.ID, &p.Name, &p.Username, &p.Email, &p.Role, &p.Team, &p.StartDate, &p.Goals, &p.Expectations)
	if errors.Is(err, sql.ErrNoRows) {
		err = apperror.ErrNotFound
	}
	return p, err
}
func (p *Repository) Intern(ctx context.Context, id string) (interns.Profile, error) {
	return scanProfile(p.db.QueryRowContext(ctx, `SELECT `+profileColumns+` FROM profiles WHERE authentik_id=$1 AND role='intern'`, id))
}
func (p *Repository) Mentor(ctx context.Context, id string) (*interns.Profile, error) {
	v, err := scanProfile(p.db.QueryRowContext(ctx, `SELECT `+profileColumns+` FROM profiles WHERE authentik_id=(SELECT mentor_id FROM assignments WHERE intern_id=$1)`, id))
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}
func (p *Repository) Interns(ctx context.Context, id string) ([]interns.Profile, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT `+profileColumns+` FROM profiles WHERE authentik_id IN (SELECT intern_id FROM assignments WHERE mentor_id=$1) ORDER BY authentik_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []interns.Profile{}
	for rows.Next() {
		v, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (p *Repository) Assigned(ctx context.Context, mentor, id string) (bool, error) {
	var ok bool
	err := p.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM assignments WHERE mentor_id=$1 AND intern_id=$2)`, mentor, id).Scan(&ok)
	return ok, err
}
