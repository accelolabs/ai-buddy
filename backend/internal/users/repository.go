package users

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Add(ctx context.Context, user User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, name, role, team) VALUES ($1, $2, $3, $4)`,
		user.ID, user.Name, user.Role, user.Team)
	if err != nil {
		return fmt.Errorf("add user %q: %w", user.ID, err)
	}

	return nil
}

// Get returns an error wrapping sql.ErrNoRows if the user does not exist.
func (r *Repository) Get(ctx context.Context, id string) (User, error) {
	var user User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, role, team FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.Name, &user.Role, &user.Team)
	if err != nil {
		return User{}, fmt.Errorf("get user %q: %w", id, err)
	}

	return user, nil
}

func (r *Repository) List(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, role, team FROM users ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer func() {
		// Read errors are reported by Scan and rows.Err below.
		_ = rows.Close()
	}()

	users := make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Role, &user.Team); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}
