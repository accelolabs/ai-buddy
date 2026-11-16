package repository

import (
	"backend/internal/tracker"
	"backend/internal/users"
	"context"
	"database/sql"
	"fmt"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Add(ctx context.Context, value tracker.Task) (tracker.Task, error) {
	err := r.db.QueryRowContext(ctx, `INSERT INTO tasks (id, user_id, description, deadline)
 VALUES ($1, $2, $3, $4) RETURNING created_at, CASE WHEN completed THEN 'completed' WHEN deadline < now() THEN 'overdue' ELSE 'pending' END`,
		value.ID, value.UserID, value.Description, value.Deadline).Scan(&value.CreatedAt, &value.Status)
	if err != nil {
		return tracker.Task{}, fmt.Errorf("add task: %w", err)
	}
	return value, nil
}

// LEFT JOIN distinguishes an accessible intern with no tasks from an inaccessible intern.
// Ownership and the current mentorship are checked in the same statement as the list.
func (r *Repository) List(ctx context.Context, user users.User, internID string) ([]tracker.Task, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT t.id, t.user_id, t.description, t.deadline, CASE WHEN completed THEN 'completed' WHEN deadline < now() THEN 'overdue' ELSE 'pending' END, t.created_at
 FROM users u LEFT JOIN tasks t ON t.user_id = u.id
 WHERE u.id = $1 AND u.role = 'intern' AND
 (($2 = 'intern' AND u.id = $3) OR ($2 = 'mentor' AND EXISTS
 (SELECT 1 FROM mentorships m WHERE m.intern_id = u.id AND m.mentor_id = $3)))
 ORDER BY t.deadline ASC, t.created_at ASC, t.id ASC`, internID, user.Role, user.ID)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()
	values := make([]tracker.Task, 0)
	accessible := false
	for rows.Next() {
		var id, userID, description sql.NullString
		var deadline, createdAt sql.NullTime
		var status string
		if err := rows.Scan(&id, &userID, &description, &deadline, &status, &createdAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		accessible = true
		if id.Valid {
			values = append(values, tracker.Task{ID: id.String, UserID: userID.String, Description: description.String, Deadline: deadline.Time, Status: status, CreatedAt: createdAt.Time})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tasks: %w", err)
	}
	if !accessible {
		return nil, fmt.Errorf("list intern tasks: %w", sql.ErrNoRows)
	}
	return values, nil
}

func (r *Repository) Complete(ctx context.Context, user users.User, id string) (tracker.Task, error) {
	var value tracker.Task
	err := r.db.QueryRowContext(ctx, `UPDATE tasks SET completed = true
 WHERE id = $1 AND user_id = $2 AND $3 = 'intern'
 RETURNING id, user_id, description, deadline, CASE WHEN completed THEN 'completed' WHEN deadline < now() THEN 'overdue' ELSE 'pending' END, created_at`, id, user.ID, user.Role).
		Scan(&value.ID, &value.UserID, &value.Description, &value.Deadline, &value.Status, &value.CreatedAt)
	if err != nil {
		return tracker.Task{}, fmt.Errorf("complete task: %w", err)
	}
	return value, nil
}
