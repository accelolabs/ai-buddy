package wiki

import (
	"ai-buddy/internal/platform/cleanup"
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUncertainUnlockDiscardsConnection(t *testing.T) {
	for _, queryFailure := range []bool{false, true} {
		t.Run(
			map[bool]string{false: "not-unlocked", true: "query-failed"}[queryFailure],
			func(t *testing.T) {
				db, m, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}
				defer cleanup.Close(db)

				db.SetMaxOpenConns(1)
				m.ExpectQuery("SELECT pg_try_advisory_lock").
					WithArgs(lockID).
					WillReturnRows(sqlmock.NewRows([]string{"acquired"}).AddRow(true))

				indexSession, err := (&Store{DB: db}).Acquire(context.Background())
				if err != nil {
					t.Fatal(err)
				}

				query := m.ExpectQuery("SELECT pg_advisory_unlock").WithArgs(lockID)

				if queryFailure {
					query.WillReturnError(errors.New("uncertain unlock"))
				} else {
					query.WillReturnRows(
						sqlmock.NewRows([]string{"unlocked"}).AddRow(false),
					)
				}
				// Raw returning ErrBadConn must close this physical connection, before DB.Close.
				m.ExpectClose()
				indexSession.Close()

				if stats := db.Stats(); stats.OpenConnections != 0 ||
					stats.Idle != 0 {
					t.Fatal("locked connection returned to pool", stats)
				}

				if err = m.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			},
		)
	}
}
