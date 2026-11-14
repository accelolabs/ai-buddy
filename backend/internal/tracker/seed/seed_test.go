package seed

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ai-buddy/internal/testutil"
)

func expectSeed(m sqlmock.Sqlmock, fail string) {
	m.ExpectBegin()
	for i := 0; i < 6; i++ {
		e := m.ExpectExec("INSERT INTO profiles")
		if fail == "profile" && i == 2 {
			e.WillReturnError(errors.New("failure"))
			m.ExpectRollback()
			return
		}

		e.WillReturnResult(sqlmock.NewResult(0, 1))
	}

	for i := 0; i < 4; i++ {
		if i < 3 {
			e := m.ExpectExec("INSERT INTO assignments")
			if fail == "assignment" {
				e.WillReturnError(errors.New("failure"))
				m.ExpectRollback()
				return
			}

			e.WillReturnResult(sqlmock.NewResult(0, 1))
		}

		for j := 0; j < 3; j++ {
			e := m.ExpectExec("INSERT INTO tasks")
			if fail == "task" {
				e.WillReturnError(errors.New("failure"))
				m.ExpectRollback()
				return
			}

			e.WillReturnResult(sqlmock.NewResult(0, 1))
		}
	}

	if fail == "commit" {
		m.ExpectCommit().WillReturnError(errors.New("failure"))
	} else {
		m.ExpectCommit()
	}
}

func TestSeed(t *testing.T) {
	srv := testutil.Authentik(t, "")
	defer srv.Close()

	for _, fail := range []string{
		"",
		"profile",
		"assignment",
		"task",
		"commit",
		"begin",
	} {
		t.Run(fail, func(t *testing.T) {
			db, m, _ := sqlmock.New()
			defer func() {
				// sqlmock cleanup after assertions; Close is not part of this scenario.
				_ = db.Close()
			}()

			if fail == "begin" {
				m.ExpectBegin().WillReturnError(errors.New("failure"))
			} else {
				expectSeed(m, fail)
			}

			e := Run(context.Background(), db, Config{URL: srv.URL, Token: "token"})
			if (e != nil) != (fail != "") {
				t.Fatal(e)
			}

			if e = m.ExpectationsWereMet(); e != nil {
				t.Fatal(e)
			}
		})
	}
}
