package cleanup

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

type closer struct {
	err   error
	calls int
}

func (c *closer) Close() error { c.calls++; return c.err }

func TestCleanup(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	defer slog.SetDefault(previous)
	c := &closer{}
	Close(c)
	if c.calls != 1 || output.Len() != 0 {
		t.Fatal(c, output.String())
	}
	c.err = errors.New("close failed")
	Close(c)
	if !strings.Contains(output.String(), "resource cleanup failed") || !strings.Contains(output.String(), "close failed") {
		t.Fatal(output.String())
	}
	db, m, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	m.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	m.ExpectCommit()
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	Rollback(tx)
	if output.Len() != 0 {
		t.Fatal(output.String())
	}
	m.ExpectBegin()
	tx, err = db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	m.ExpectRollback().WillReturnError(errors.New("rollback failed"))
	Rollback(tx)
	if !strings.Contains(output.String(), "rollback failed") {
		t.Fatal(output.String())
	}
	if err = m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
