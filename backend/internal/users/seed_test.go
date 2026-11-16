package users

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

const seedSQL0 = `
				INSERT INTO users (id, name, role, team)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (id) DO NOTHING
			`

const seedSQL1 = `
				INSERT INTO mentorships (intern_id, mentor_id)
				VALUES ($1, $2)
				ON CONFLICT (intern_id) DO NOTHING
			`

func TestSeed(t *testing.T) {
	const (
		noFailure  = -3
		failBegin  = -2
		failCommit = -1
	)
	failure := errors.New("seed failure")
	records := []struct {
		query string
		args  []driver.Value
	}{}

	for _, user := range []User{
		{ID: "intern.backend.1", Name: "Фёдор Добронравов", Role: RoleIntern, Team: "backend"},
		{ID: "intern.backend.2", Name: "Эдуард Радзюкевич", Role: RoleIntern, Team: "backend"},
		{ID: "mentor.backend", Name: "Сергей Дорогов", Role: RoleMentor, Team: "backend"},
		{ID: "intern.platform.1", Name: "Ирина Медведева", Role: RoleIntern, Team: "platform"},
		{ID: "intern.platform.2", Name: "Галина Данилова", Role: RoleIntern, Team: "platform"},
		{ID: "mentor.platform", Name: "Андрей Кайков", Role: RoleMentor, Team: "platform"},
	} {
		records = append(records, struct {
			query string
			args  []driver.Value
		}{seedSQL0, []driver.Value{user.ID, user.Name, user.Role, user.Team}})
	}

	for _, team := range []string{"backend", "platform"} {
		for i := 1; i <= 2; i++ {
			records = append(records, struct {
				query string
				args  []driver.Value
			}{
				seedSQL1,
				[]driver.Value{fmt.Sprintf("intern.%s.%d", team, i), "mentor." + team},
			})
		}
	}

	// Every insert can fail; later statements must not execute after that failure.
	for fail := noFailure; fail < len(records); fail++ {
		t.Run(fmt.Sprintf("failure_%d", fail), func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}

			defer func() { _ = db.Close() }()
			begin := mock.ExpectBegin()
			if fail == failBegin {
				begin.WillReturnError(failure)
			} else {
				for i, record := range records {
					e := mock.ExpectExec(regexp.QuoteMeta(record.query)).WithArgs(record.args...)
					if i == fail {
						e.WillReturnError(failure)
						break
					}

					e.WillReturnResult(sqlmock.NewResult(0, 1))
				}

				if fail >= 0 {
					mock.ExpectRollback()
				} else {
					commit := mock.ExpectCommit()
					if fail == failCommit {
						commit.WillReturnError(failure)
					}
				}
			}

			err = Seed(t.Context(), db)
			if fail == noFailure {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, failure) {
				t.Fatalf("%v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
