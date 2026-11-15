package users

import (
	"backend-ai/internal/platform/database"
	"context"
	"database/sql"
	"errors"
)

//nolint:nilnil // An unassigned mentor is an expected optional relation.
func (p *Store) Mentor(ctx context.Context, id string) (*User, error) {
	const query = `
		SELECT ` + columns + `
		FROM profiles
		WHERE id=(
			SELECT mentor_id FROM assignments WHERE intern_id=$1
		)
	`

	row := p.DB.QueryRowContext(ctx, query, id)

	mentor, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &mentor, nil
}

func (p *Store) Interns(ctx context.Context, id string) ([]User, error) {
	const query = `
		SELECT ` + columns + `
		FROM profiles
		WHERE id IN (
			SELECT intern_id FROM assignments WHERE mentor_id=$1
		)
		ORDER BY id
	`

	return database.Query(ctx, p.DB, query, scan, id)
}

func (p *Store) Assigned(ctx context.Context, mentor, id string) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM assignments WHERE mentor_id=$1 AND intern_id=$2
		)
	`

	var assigned bool

	err := p.DB.QueryRowContext(ctx, query, mentor, id).Scan(&assigned)

	return assigned, err
}
