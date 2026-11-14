package main

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSelection(t *testing.T) {
	original, args := openDatabase, os.Args
	defer func() { openDatabase = original; os.Args = args }()
	t.Setenv("DATABASE_URL", "postgres://selected/app")
	for _, service := range []string{"tracker", "ticket"} {
		for _, command := range []string{"up", "down", "status"} {
			op, err := prepare(service, command)
			if err != nil {
				t.Fatal(err)
			}
			db, m, _ := sqlmock.New()
			defer db.Close()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if op(ctx, db) == nil {
				t.Fatal("cancelled migration succeeded")
			}
			if err := m.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		}
	}
	os.Args = []string{"migrate", "tracker"}
	db, m, _ := sqlmock.New()
	m.ExpectClose()
	openDatabase = func(context.Context, string) (*sql.DB, error) { return db, nil }
	if run() == nil {
		t.Fatal("unavailable migration database accepted")
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
