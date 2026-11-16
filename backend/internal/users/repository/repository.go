package users

import (
	"context"
	"database/sql"
	"fmt"

	// Project packages.
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

// Get returns an error wrapping sql.ErrNoRows if the user does not exist.
func (r *Repository) Get(ctx context.Context, id string) (users.User, error) {
	var user users.User
	err := r.db.QueryRowContext(
		ctx,
		`
			SELECT id, name, role, team
			FROM users
			WHERE id = $1
		`,
		id,
	).Scan(&user.ID, &user.Name, &user.Role, &user.Team)
	if err != nil {
		return users.User{}, fmt.Errorf("get user %q: %w", id, err)
	}

	return user, nil
}

func (r *Repository) List(ctx context.Context) ([]users.User, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
			SELECT id, name, role, team
			FROM users
			ORDER BY id
		`,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	defer func() {
		// Read errors are reported by Scan and rows.Err below.
		_ = rows.Close()
	}()

	list := make([]users.User, 0)

	for rows.Next() {
		var user users.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Role, &user.Team); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}

		list = append(list, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return list, nil
}
