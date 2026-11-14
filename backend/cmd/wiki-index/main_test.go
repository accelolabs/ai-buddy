package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRun(t *testing.T) {
	orig := openDatabase
	defer func() { openDatabase = orig }()
	ctx := context.Background()
	out := &bytes.Buffer{}
	for _, args := range [][]string{{"--bad"}, {"positional"}} {
		if run(ctx, args, out) == nil {
			t.Fatal("accepted args", args)
		}
	}
	t.Setenv("DATABASE_URL", "")
	if run(ctx, nil, out) == nil {
		t.Fatal("accepted config")
	}
	t.Setenv("DATABASE_URL", "postgres://localhost/wiki")
	openDatabase = func(context.Context, string) (*sql.DB, error) { return nil, errors.New("down") }
	if run(ctx, nil, out) == nil {
		t.Fatal("accepted db")
	}
	db, m, _ := sqlmock.New()
	m.ExpectQuery("SELECT pg_try_advisory_lock").WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(false))
	m.ExpectClose()
	openDatabase = func(context.Context, string) (*sql.DB, error) { return db, nil }
	if run(ctx, []string{"--full"}, out) == nil {
		t.Fatal("accepted competing index")
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
