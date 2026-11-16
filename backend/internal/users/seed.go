package users

import (
	"context"
	"database/sql"
	"fmt"
)

// Seed adds demo users and mentorships without replacing existing records.
func Seed(ctx context.Context, db *sql.DB) error {
	demoUsers := []User{
		{
			ID:   "intern.backend.1",
			Name: "Фёдор Добронравов",
			Role: RoleIntern,
			Team: "backend",
		},
		{
			ID:   "intern.backend.2",
			Name: "Эдуард Радзюкевич",
			Role: RoleIntern,
			Team: "backend",
		},
		{
			ID:   "mentor.backend",
			Name: "Сергей Дорогов",
			Role: RoleMentor,
			Team: "backend",
		},
		{
			ID:   "intern.platform.1",
			Name: "Ирина Медведева",
			Role: RoleIntern,
			Team: "platform",
		},
		{
			ID:   "intern.platform.2",
			Name: "Галина Данилова",
			Role: RoleIntern,
			Team: "platform",
		},
		{
			ID:   "mentor.platform",
			Name: "Андрей Кайков",
			Role: RoleMentor,
			Team: "platform",
		},
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin user seed: %w", err)
	}

	defer func() {
		// Rollback is a no-op after a successful commit.
		_ = tx.Rollback()
	}()

	for _, user := range demoUsers {
		_, err := tx.ExecContext(
			ctx,
			`
				INSERT INTO users (id, name, role, team)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (id) DO NOTHING
			`,
			user.ID,
			user.Name,
			user.Role,
			user.Team,
		)
		if err != nil {
			return fmt.Errorf("seed user %q: %w", user.ID, err)
		}
	}

	for _, link := range []Mentorship{
		{
			InternID: "intern.backend.1",
			MentorID: "mentor.backend",
		},
		{
			InternID: "intern.backend.2",
			MentorID: "mentor.backend",
		},
		{
			InternID: "intern.platform.1",
			MentorID: "mentor.platform",
		},
		{
			InternID: "intern.platform.2",
			MentorID: "mentor.platform",
		},
	} {
		_, err := tx.ExecContext(
			ctx,
			`
				INSERT INTO mentorships (intern_id, mentor_id)
				VALUES ($1, $2)
				ON CONFLICT (intern_id) DO NOTHING
			`,
			link.InternID,
			link.MentorID,
		)
		if err != nil {
			return fmt.Errorf("seed mentorship for %q: %w", link.InternID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit user seed: %w", err)
	}

	return nil
}
