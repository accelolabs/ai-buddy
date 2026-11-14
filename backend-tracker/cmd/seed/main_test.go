package main

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRun(t *testing.T) {
	orig := openDatabase
	defer func() {
		openDatabase = orig
	}()

	t.Setenv("DATABASE_URL", "")
	if run() == nil {
		t.Fatal("invalid config")
	}

	t.Setenv("DATABASE_URL", "postgres://db/app")
	t.Setenv("AUTHENTIK_URL", "http://127.0.0.1:1")
	t.Setenv("AUTHENTIK_API_TOKEN", "seed-token")
	openDatabase = func(context.Context, string) (*sql.DB, error) {
		return nil, errors.New("down")
	}
	if run() == nil {
		t.Fatal("database failure")
	}

	db, m, _ := sqlmock.New()
	m.ExpectClose()
	openDatabase = func(context.Context, string) (*sql.DB, error) {
		return db, nil
	}
	if err := run(); err == nil {
		t.Fatal("expected API failure")
	}

	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
