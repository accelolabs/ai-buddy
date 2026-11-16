package users

import (
	"context"
	"fmt"

	// Project packages.
	"backend/internal/users"
)

func (r *Repository) ListInterns(ctx context.Context, mentorID string) ([]users.User, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
			SELECT u.id, u.name, u.role, u.team
			FROM users AS u
			JOIN mentorships AS m ON m.intern_id = u.id
			WHERE m.mentor_id = $1
			ORDER BY u.id
		`,
		mentorID,
	)
	if err != nil {
		return nil, fmt.Errorf("list mentor interns: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	interns := make([]users.User, 0)

	for rows.Next() {
		var user users.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Role, &user.Team); err != nil {
			return nil, fmt.Errorf("scan mentor intern: %w", err)
		}

		interns = append(interns, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read mentor interns: %w", err)
	}

	return interns, nil
}

func (r *Repository) GetMentor(ctx context.Context, internID string) (users.User, error) {
	var user users.User
	err := r.db.QueryRowContext(
		ctx,
		`
			SELECT u.id, u.name, u.role, u.team
			FROM users u
			JOIN mentorships m ON m.mentor_id = u.id
			WHERE m.intern_id = $1
		`,
		internID,
	).
		Scan(&user.ID, &user.Name, &user.Role, &user.Team)
	if err != nil {
		return users.User{}, fmt.Errorf("get mentor: %w", err)
	}

	return user, nil
}
