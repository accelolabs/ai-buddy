package users

import (
	"backend/internal/users"
	"context"
	"fmt"
)

// AddMentorship links an intern to a mentor, checking their current roles.
func (r *Repository) AddMentorship(ctx context.Context, link users.Mentorship) error {
	result, err := r.db.ExecContext(ctx, `
  INSERT INTO mentorships (intern_id, mentor_id)
  SELECT i.id, m.id FROM users AS i CROSS JOIN users AS m
  WHERE i.id = $1 AND i.role = 'intern' AND m.id = $2 AND m.role = 'mentor'`, link.InternID, link.MentorID)
	if err != nil {
		return fmt.Errorf("add mentorship: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read mentorship insert result: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("mentorship requires an existing intern and mentor")
	}
	return nil
}

func (r *Repository) ListInterns(ctx context.Context, mentorID string) ([]users.User, error) {
	rows, err := r.db.QueryContext(ctx, `
  SELECT u.id, u.name, u.role, u.team
  FROM users AS u JOIN mentorships AS m ON m.intern_id = u.id
  WHERE m.mentor_id = $1 ORDER BY u.id`, mentorID)
	if err != nil {
		return nil, fmt.Errorf("list mentor interns: %w", err)
	}
	defer func() { _ = rows.Close() }()
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
