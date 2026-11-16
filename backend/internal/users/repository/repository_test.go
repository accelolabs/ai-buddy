package users

import (
	"database/sql"
	"errors"
	"reflect"
	"regexp"
	"testing"

	// Project packages.
	"backend/internal/users"

	"github.com/DATA-DOG/go-sqlmock"
)

const getSQL = `
			SELECT id, name, role, team
			FROM users
			WHERE id = $1
		`

const listSQL = `
			SELECT id, name, role, team
			FROM users
			ORDER BY id
		`

const internsSQL = `
			SELECT u.id, u.name, u.role, u.team
			FROM users AS u
			JOIN mentorships AS m ON m.intern_id = u.id
			WHERE m.mentor_id = $1
			ORDER BY u.id
		`

const mentorSQL = `
			SELECT u.id, u.name, u.role, u.team
			FROM users u
			JOIN mentorships m ON m.mentor_id = u.id
			WHERE m.intern_id = $1
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

func TestReads(t *testing.T) {
	expected := users.User{ID: "intern", Name: "Name", Role: "intern", Team: "backend"}
	columns := []string{"id", "name", "role", "team"}

	for _, mentor := range []bool{false, true} {
		for _, mode := range []string{"ok", "missing", "query", "scan"} {
			t.Run(fmtName(mentor, mode), func(t *testing.T) {
				repo, mock := mockRepository(t)
				query := getSQL
				if mentor {
					query = mentorSQL
				}

				e := mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs("intern")
				switch mode {
				case "query":
					e.WillReturnError(errQueryFailure)
				case "missing":
					e.WillReturnRows(sqlmock.NewRows(columns))
				case "scan":
					e.WillReturnRows(sqlmock.NewRows(columns).AddRow(nil, "Name", "intern", "backend"))
				default:
					e.WillReturnRows(sqlmock.NewRows(columns).AddRow("intern", "Name", "intern", "backend"))
				}

				var got users.User
				var err error
				if mentor {
					got, err = repo.GetMentor(t.Context(), "intern")
				} else {
					got, err = repo.Get(t.Context(), "intern")
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

	for _, interns := range []bool{false, true} {
		for _, mode := range []string{"ok", "empty", "query", "scan", "iterate"} {
			t.Run(fmtName(interns, mode), func(t *testing.T) {
				repo, mock := mockRepository(t)
				query := listSQL
				if interns {
					query = internsSQL
				}

				e := mock.ExpectQuery(regexp.QuoteMeta(query))
				if interns {
					e.WithArgs("mentor")
				}

				rows := sqlmock.NewRows(columns)
				switch mode {
				case "ok":
					rows.AddRow("intern", "Name", "intern", "backend")
				case "scan":
					rows.AddRow(nil, "Name", "intern", "backend")
				case "iterate":
					rows.AddRow("intern", "Name", "intern", "backend").RowError(0, errQueryFailure)
				}

				if mode == "query" {
					e.WillReturnError(errQueryFailure)
				} else {
					e.WillReturnRows(rows).RowsWillBeClosed()
				}

				var got []users.User
				var err error
				if interns {
					got, err = repo.ListInterns(t.Context(), "mentor")
				} else {
					got, err = repo.List(t.Context())
				}

				switch mode {
				case "ok":
					if err != nil || !reflect.DeepEqual(got, []users.User{expected}) {
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
	}
}

func fmtName(flag bool, mode string) string {
	if flag {
		return "related/" + mode
	}

	return "direct/" + mode
}
