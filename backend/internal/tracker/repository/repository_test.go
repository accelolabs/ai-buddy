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
	"backend/internal/tracker"
	"backend/internal/users"

	"github.com/DATA-DOG/go-sqlmock"
)

const addSQL = `
			INSERT INTO tasks (id, user_id, description, deadline)
			VALUES ($1, $2, $3, $4)
			RETURNING created_at,
				CASE
					WHEN completed THEN 'completed'
					WHEN deadline < now() THEN 'overdue'
					ELSE 'pending'
				END
		`

const listSQL = `
			SELECT t.id, t.user_id, t.description, t.deadline,
				CASE
					WHEN completed THEN 'completed'
					WHEN deadline < now() THEN 'overdue'
					ELSE 'pending'
				END, t.created_at
			FROM users u
			LEFT JOIN tasks t ON t.user_id = u.id
			WHERE u.id = $1
				AND u.role = 'intern'
				AND (
					($2 = 'intern' AND u.id = $3)
					OR (
						$2 = 'mentor'
						AND EXISTS (
							SELECT 1
							FROM mentorships m
							WHERE m.intern_id = u.id
								AND m.mentor_id = $3
						)
					)
				)
			ORDER BY t.deadline ASC, t.created_at ASC, t.id ASC
		`

const completeSQL = `
			UPDATE tasks
			SET completed = true
			WHERE id = $1
				AND user_id = $2
				AND $3 = 'intern'
			RETURNING id, user_id, description, deadline,
				CASE
					WHEN completed THEN 'completed'
					WHEN deadline < now() THEN 'overdue'
					ELSE 'pending'
				END, created_at
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
	expected := tracker.Task{
		ID:          "id",
		UserID:      "intern",
		Description: "Body",
		CreatedAt:   now,
		Deadline:    now,
		Status:      "pending",
	}
	columns := []string{"id", "user_id", "description", "deadline", "status", "created_at"}

	for _, mode := range []string{"ok", "empty", "query", "scan", "iterate"} {
		t.Run("list/"+mode, func(t *testing.T) {
			repo, mock := mockRepository(t)
			e := mock.ExpectQuery(regexp.QuoteMeta(listSQL)).WithArgs("intern", user.Role, user.ID)
			rows := sqlmock.NewRows(columns)
			switch mode {
			case "ok":
				rows.AddRow("id", "intern", "Body", now, "pending", now)
			case "scan":
				rows.AddRow("id", "intern", "Body", "bad-time", "pending", now)
			case "iterate":
				rows.AddRow("id", "intern", "Body", now, "pending", now).RowError(0, errQueryFailure)
			}

			if mode == "query" {
				e.WillReturnError(errQueryFailure)
			} else {
				e.WillReturnRows(rows).RowsWillBeClosed()
			}

			got, err := repo.List(t.Context(), user, "intern")
			switch mode {
			case "ok":
				if err != nil || !reflect.DeepEqual(got, []tracker.Task{expected}) {
					t.Fatalf("%+v %v", got, err)
				}
			case "empty":
				if !errors.Is(err, sql.ErrNoRows) {
					t.Fatal(err)
				}
			default:
				if err == nil {
					t.Fatal("missing error")
				}
			}
		})
	}

	for _, operation := range []string{"add", "complete"} {
		for _, mode := range []string{"ok", "missing", "query", "scan"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				repo, mock := mockRepository(t)
				query := completeSQL
				args := []driver.Value{"id", user.ID, user.Role}
				rowColumns := columns
				rowValues := []driver.Value{"id", "intern", "Body", now, "pending", now}
				if operation == "add" {
					query = addSQL
					args = []driver.Value{"id", "intern", "Body", now}
					rowColumns = []string{"created_at", "status"}
					rowValues = []driver.Value{now, "pending"}
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

				var got tracker.Task
				var err error
				if operation == "add" {
					got, err = repo.Add(t.Context(), expected)
				} else {
					got, err = repo.Complete(t.Context(), user, "id")
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

func TestAccessibleEmptyIntern(t *testing.T) {
	repo, mock := mockRepository(t)
	mock.ExpectQuery(regexp.QuoteMeta(listSQL)).WithArgs("intern", "mentor", "mentor").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "description", "deadline", "status", "created_at"}).
			AddRow(nil, nil, nil, nil, "pending", nil)).RowsWillBeClosed()
	got, err := repo.List(t.Context(), users.User{ID: "mentor", Role: "mentor"}, "intern")
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("%+v %v", got, err)
	}
}
