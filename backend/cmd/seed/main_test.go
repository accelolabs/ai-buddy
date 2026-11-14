package main

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSelectionAndConfig(t *testing.T) {
	original, args := openDatabase, os.Args
	defer func() { openDatabase = original; os.Args = args }()
	t.Setenv("DATABASE_URL", "postgres://selected/app")
	t.Setenv("AUTHENTIK_URL", "http://127.0.0.1:1")
	t.Setenv("TRACKER_URL", "http://tracker")
	t.Setenv("TICKET_API_KEY", "ticket")
	for _, service := range []string{"tracker", "ticket"} {
		t.Setenv("AUTHENTIK_API_TOKEN", "")
		if _, err := prepare(service, ""); err == nil {
			t.Fatal("missing seed token")
		}
		t.Setenv("AUTHENTIK_API_TOKEN", "token")
		os.Args = []string{"seed", service}
		db, m, _ := sqlmock.New()
		m.ExpectClose()
		openDatabase = func(context.Context, string) (*sql.DB, error) { return db, nil }
		if run() == nil {
			t.Fatal("unavailable Authentik accepted")
		}
		if err := m.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	}
}
