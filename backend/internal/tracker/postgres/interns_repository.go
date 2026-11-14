package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"ai-buddy/internal/tracker/apperror"
	"ai-buddy/internal/tracker/interns"
)

const profileColumns = `
	authentik_id,
	name,
	username,
	email,
	role,
	team,
	to_char(start_date,'YYYY-MM-DD'),
	goals,
	expectations
`

func scanProfile(row scanner) (interns.Profile, error) {
	var profile interns.Profile
	err := row.Scan(
		&profile.ID,
		&profile.Name,
		&profile.Username,
		&profile.Email,
		&profile.Role,
		&profile.Team,
		&profile.StartDate,
		&profile.Goals,
		&profile.Expectations,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = apperror.ErrNotFound
	}

	return profile, err
}

func (p *Repository) Intern(ctx context.Context, id string) (interns.Profile, error) {
	const query = `
		SELECT ` + profileColumns + `
		FROM profiles
		WHERE authentik_id=$1 AND role='intern'
	`

	row := p.db.QueryRowContext(ctx, query, id)
	return scanProfile(row)
}

func (p *Repository) Mentor(ctx context.Context, id string) (*interns.Profile, error) {
	const query = `
		SELECT ` + profileColumns + `
		FROM profiles
		WHERE authentik_id=(
			SELECT mentor_id FROM assignments WHERE intern_id=$1
		)
	`

	row := p.db.QueryRowContext(ctx, query, id)
	mentor, err := scanProfile(row)
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &mentor, nil
}

func (p *Repository) Interns(ctx context.Context, id string) ([]interns.Profile, error) {
	const query = `
		SELECT ` + profileColumns + `
		FROM profiles
		WHERE authentik_id IN (
			SELECT intern_id FROM assignments WHERE mentor_id=$1
		)
		ORDER BY authentik_id
	`

	rows, err := p.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Warn("resource cleanup failed", "error", err)
		}
	}()

	profiles := []interns.Profile{}
	for rows.Next() {
		profile, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}

		profiles = append(profiles, profile)
	}

	return profiles, rows.Err()
}

func (p *Repository) Assigned(ctx context.Context, mentor, id string) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM assignments WHERE mentor_id=$1 AND intern_id=$2
		)
	`

	var assigned bool
	err := p.db.QueryRowContext(ctx, query, mentor, id).Scan(&assigned)
	return assigned, err
}
