package repository

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"regexp"
	"testing"
	"time"

	// Project packages.
	"backend/internal/ticket"
	"backend/internal/users"

	"github.com/DATA-DOG/go-sqlmock"
)

const addSQL = `
			INSERT INTO tickets (id, user_id, title, description)
			VALUES ($1, $2, $3, $4)
			RETURNING created_at
		`

const listSQL = `
			SELECT t.id, t.user_id, t.title, t.description, t.created_at
			FROM tickets t
			WHERE (
				($1 = 'intern' AND t.user_id = $2)
				OR (
					$1 = 'mentor'
					AND EXISTS (
						SELECT 1
						FROM mentorships m
						WHERE m.intern_id = t.user_id
							AND m.mentor_id = $2
					)
				)
			)
			ORDER BY t.created_at DESC, t.id DESC
		`

const getSQL = `
			SELECT t.id, t.user_id, t.title, t.description, t.created_at
			FROM tickets t
			WHERE (
				($1 = 'intern' AND t.user_id = $2)
				OR (
					$1 = 'mentor'
					AND EXISTS (
						SELECT 1
						FROM mentorships m
						WHERE m.intern_id = t.user_id
							AND m.mentor_id = $2
					)
				)
			)
				AND t.id = $3
		`

func mockRepository(t *testing.T) (*Repository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}

		_ = db.Close()
	})

	return NewRepository(db), mock
}

var errQueryFailure = errors.New("query failure")

func TestRepository(t *testing.T) {
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	user := users.User{ID: "intern", Role: users.RoleIntern}
	expected := ticket.Ticket{ID: "id", UserID: "intern", Description: "Body", CreatedAt: now, Title: "Title"}
	columns := []string{"id", "user_id", "title", "description", "created_at"}

	for _, mode := range []string{"ok", "empty", "query", "scan", "iterate"} {
		t.Run("list/"+mode, func(t *testing.T) {
			repo, mock := mockRepository(t)
			e := mock.ExpectQuery(regexp.QuoteMeta(listSQL)).WithArgs(user.Role, user.ID)
			rows := sqlmock.NewRows(columns)
			switch mode {
			case "ok":
				rows.AddRow("id", "intern", "Title", "Body", now)
			case "scan":
				rows.AddRow(nil, "intern", "Title", "Body", now)
			case "iterate":
				rows.AddRow("id", "intern", "Title", "Body", now).RowError(0, errQueryFailure)
			}

			if mode == "query" {
				e.WillReturnError(errQueryFailure)
			} else {
				e.WillReturnRows(rows).RowsWillBeClosed()
			}

			got, err := repo.List(t.Context(), user)
			switch mode {
			case "ok":
				if err != nil || !reflect.DeepEqual(got, []ticket.Ticket{expected}) {
					t.Fatalf("%+v %v", got, err)
				}
			case "empty":
				if err != nil || got == nil || len(got) != 0 {
					t.Fatalf("%+v %v", got, err)
				}
			default:
				if err == nil {
					t.Fatal("missing error")
				}
			}
		})
	}

	for _, operation := range []string{"add", "get"} {
		for _, mode := range []string{"ok", "missing", "query", "scan"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				repo, mock := mockRepository(t)
				query := getSQL
				args := []driver.Value{user.Role, user.ID, "id"}
				rowColumns := columns
				rowValues := []driver.Value{"id", "intern", "Title", "Body", now}
				if operation == "add" {
					query = addSQL
					args = []driver.Value{"id", "intern", "Title", "Body"}
					rowColumns = []string{"created_at"}
					rowValues = []driver.Value{now}
				}

				e := mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(args...)
				rows := sqlmock.NewRows(rowColumns)
				if mode == "ok" {
					rows.AddRow(rowValues...)
				}

				if mode == "scan" {
					rowValues[0] = nil
					rows.AddRow(rowValues...)
				}

				if mode == "query" {
					e.WillReturnError(errQueryFailure)
				} else {
					e.WillReturnRows(rows).RowsWillBeClosed()
				}

				var got ticket.Ticket
				var err error
				if operation == "add" {
					got, err = repo.Add(t.Context(), expected)
				} else {
					got, err = repo.Get(t.Context(), user, "id")
				}

				if mode == "ok" {
					if err != nil || got != expected {
						t.Fatalf("%+v %v", got, err)
					}
				} else if err == nil {
					t.Fatal("missing error")
				}

				if mode == "missing" && !errors.Is(err, sql.ErrNoRows) {
					t.Fatal(err)
				}

				if mode == "query" && !errors.Is(err, errQueryFailure) {
					t.Fatal(err)
				}
			})
		}
	}
}
