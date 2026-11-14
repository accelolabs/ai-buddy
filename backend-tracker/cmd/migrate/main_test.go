package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRun(t *testing.T) {
	orig := openDatabase
	args := os.Args
	defer func() {
		openDatabase = orig
		os.Args = args
	}()

	t.Setenv("DATABASE_URL", "")
	if run() == nil {
		t.Fatal("invalid config")
	}

	t.Setenv("DATABASE_URL", "postgres://db/app")
	openDatabase = func(context.Context, string) (*sql.DB, error) {
		return nil, errors.New("down")
	}
	if run() == nil {
		t.Fatal("database failure")
	}

	for _, argv := range [][]string{{"migrate"}, {"migrate", "invalid"}} {
		os.Args = argv
		db, m, _ := sqlmock.New()
		m.ExpectClose()
		openDatabase = func(context.Context, string) (*sql.DB, error) {
			return db, nil
		}
		if run() == nil {
			t.Fatal("expected migration failure")
		}

		if err := m.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	}
}
