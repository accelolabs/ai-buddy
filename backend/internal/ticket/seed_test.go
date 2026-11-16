package ticket

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

const seedSQL0 = `
					INSERT INTO tickets (id, user_id, title, description)
					VALUES ($1, $2, $3, $4)
					ON CONFLICT (id) DO NOTHING
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

	for _, intern := range []string{"intern.backend.1", "intern.backend.2", "intern.platform.1", "intern.platform.2"} {
		for i, record := range []struct{ title, description string }{
			{"Доступ к репозиторию", "Прошу предоставить доступ к репозиторию команды для выполнения задач стажировки."},
			{"Локальный запуск", "Подскажите порядок локального запуска приложения и необходимые настройки."},
		} {
			records = append(records, struct {
				query string
				args  []driver.Value
			}{
				seedSQL0,
				[]driver.Value{fmt.Sprintf("demo.ticket.%s.%d", intern, i+1), intern, record.title, record.description},
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
