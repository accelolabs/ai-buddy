package buddy

import (
	"backend-ai/internal/platform/cleanup"
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestHistoryResourceAndTransactionFailures(t *testing.T) {
	fault := errors.New("driver failure")

	columns := [][]string{
		{"id", "chat_id", "run_id", "role", "content", "created_at"},
		{"id", "chat_id", "owner_id", "message_id", "status", "error", "created_at"},
		{"id", "run_id", "owner_id", "kind", "status", "fields", "result", "error"},
		{"run_id", "kind", "data"},
	}
	for failure := 0; failure <= len(columns); failure++ {
		for _, kind := range []string{"scan", "iteration", "query"} {
			t.Run(fmt.Sprintf("%d/%s", failure, kind), func(t *testing.T) {
				db, m, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}
				defer cleanup.Close(db)

				m.ExpectBegin()
				m.ExpectQuery("SELECT id,owner_id,title,created_at FROM chats").
					WithArgs("chat", "owner").
					WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "title", "created_at"}).AddRow("chat", "owner", "title", time.Now()))

				for i, cols := range columns {
					query := m.ExpectQuery("SELECT").WithArgs("chat")

					if i == failure && kind == "query" {
						query.WillReturnError(fault)
						break
					}

					rows := sqlmock.NewRows(cols)

					if i == failure {
						if kind == "scan" {
							rows = sqlmock.NewRows([]string{"bad"}).
								AddRow("bad")
						} else {
							rows.AddRow(make([]driver.Value, len(cols))...).
								RowError(0, fault)
						}
					}

					query.WillReturnRows(rows).RowsWillBeClosed()

					if i == failure {
						break
					}
				}

				if failure == len(columns) {
					m.ExpectCommit().WillReturnError(fault)
				} else {
					m.ExpectRollback()
				}

				if _, err = (&Store{DB: db}).History(
					context.Background(),
					"owner",
					"chat",
				); err == nil {
					t.Fatal("lost driver error")
				}

				if err = m.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
