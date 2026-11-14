package cli

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestInvalidArgumentsBeforeDependencies(t *testing.T) {
	for _, tc := range []struct {
		args      []string
		migration bool
	}{
		{nil, true}, {[]string{"up"}, true}, {[]string{"wiki"}, false},
		{[]string{"tracker", "invalid"}, true}, {[]string{"ticket", "up", "extra"}, true},
		{[]string{"tracker", "extra"}, false},
	} {
		called := false
		prepare := func(string, string) (Operation, error) { called = true; return nil, nil }
		if Run(tc.args, tc.migration, prepare, nil) == nil || called {
			t.Fatal(tc)
		}
	}
}

func TestRun(t *testing.T) {
	for _, fail := range []string{"", "prepare", "config", "open", "operation"} {
		t.Run(fail, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://selected/app")
			if fail == "config" {
				t.Setenv("DATABASE_URL", "")
			}
			want := errors.New("failure")
			db, m, _ := sqlmock.New()
			defer db.Close()
			called := false
			prepare := func(service, command string) (Operation, error) {
				if service != "ticket" || command != "up" {
					t.Fatal(service, command)
				}
				if fail == "prepare" {
					return nil, want
				}
				return func(ctx context.Context, got *sql.DB) error {
					called = true
					if got != db {
						t.Fatal("wrong database")
					}
					if _, ok := ctx.Deadline(); !ok {
						t.Fatal("unbounded operation")
					}
					if fail == "operation" {
						return want
					}
					return nil
				}, nil
			}
			open := func(_ context.Context, url string) (*sql.DB, error) {
				if url != "postgres://selected/app" {
					t.Fatal("wrong DATABASE_URL")
				}
				if fail == "open" {
					return nil, want
				}
				m.ExpectClose()
				return db, nil
			}
			err := Run([]string{"ticket"}, true, prepare, open)
			if (err != nil) != (fail != "") || called != (fail == "" || fail == "operation") {
				t.Fatal(err, called)
			}
			if err := m.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, command := range []string{"up", "down", "status"} {
		t.Setenv("DATABASE_URL", "postgres://selected/app")
		called := false
		prepare := func(service, got string) (Operation, error) {
			called = true
			if service != "tracker" || got != command {
				t.Fatal(service, got)
			}
			return nil, errors.New("stop")
		}
		if Run([]string{"tracker", command}, true, prepare, nil) == nil || !called {
			t.Fatal(command)
		}
	}
}
