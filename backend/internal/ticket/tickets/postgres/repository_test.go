package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"ai-buddy/internal/ticket/apperror"
	"ai-buddy/internal/ticket/tickets"
)

func ticketRows(q string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "sender_id", "recipient_id", "question", "description", "created_at"}).AddRow("id", "s", "m", q, "d", time.Now())
}
func TestCreate(t *testing.T) {
	for _, mode := range []string{"new", "repeat", "conflict", "insert error", "read error"} {
		t.Run(mode, func(t *testing.T) {
			db, m, _ := sqlmock.New()
			defer func() {
				// sqlmock cleanup after assertions; Close is not part of this scenario.
				_ = db.Close()
			}()
			repo := New(db)
			insert := m.ExpectQuery("INSERT INTO tickets").WithArgs(sqlmock.AnyArg(), "s", "m", "q", "d", "k")
			switch mode {
			case "insert error":
				insert.WillReturnError(errors.New("down"))
			case "new":
				insert.WillReturnRows(ticketRows("q"))
			default:
				insert.WillReturnRows(sqlmock.NewRows([]string{"id"}))
				find := m.ExpectQuery("SELECT").WithArgs("s", "k")
				switch mode {
				case "read error":
					find.WillReturnError(errors.New("down"))
				case "conflict":
					find.WillReturnRows(ticketRows("other"))
				default:
					find.WillReturnRows(ticketRows("q"))
				}
			}
			got, created, err := repo.Create(context.Background(), "s", "m", "k", tickets.CreateInput{Question: "q", Description: "d"})
			if mode == "new" || mode == "repeat" {
				if err != nil || created != (mode == "new") || got.RecipientID != "m" {
					t.Fatal(got, created, err)
				}
			} else if err == nil {
				t.Fatal("expected failure")
			}
			if mode == "conflict" && !errors.Is(err, apperror.ErrConflict) {
				t.Fatal(err)
			}
			if err := m.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestFindAndLists(t *testing.T) {
	db, m, _ := sqlmock.New()
	defer func() {
		// sqlmock cleanup after assertions; Close is not part of this scenario.
		_ = db.Close()
	}()
	repo := New(db)
	ctx := context.Background()
	m.ExpectQuery("SELECT").WithArgs("s", "missing").WillReturnError(sql.ErrNoRows)
	if _, err := repo.Find(ctx, "s", "missing"); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatal(err)
	}
	for _, list := range []struct {
		fn     func(context.Context, string) ([]tickets.Ticket, error)
		column string
	}{{repo.Sent, "sender_id"}, {repo.Incoming, "recipient_id"}} {
		for _, mode := range []string{"rows", "empty", "query error", "scan error", "rows error"} {
			e := m.ExpectQuery("WHERE " + list.column + "=\\$1\\s+ORDER BY created_at,id").WithArgs("s")
			switch mode {
			case "rows":
				e.WillReturnRows(ticketRows("q"))
			case "empty":
				e.WillReturnRows(sqlmock.NewRows([]string{"id"}))
			case "query error":
				e.WillReturnError(errors.New("down"))
			case "scan error":
				e.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("bad"))
			case "rows error":
				e.WillReturnRows(ticketRows("q").RowError(0, errors.New("down")))
			}
			got, err := list.fn(ctx, "s")
			if mode == "rows" {
				if err != nil || len(got) != 1 {
					t.Fatal(got, err)
				}
			} else if mode == "empty" {
				if err != nil || got == nil || len(got) != 0 {
					t.Fatal(got, err)
				}
			} else if err == nil {
				t.Fatal("expected error", mode)
			}
		}
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
