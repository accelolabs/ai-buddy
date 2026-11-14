package migrations

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCommandsAndErrors(t *testing.T) {
	if Run(context.Background(), nil, "up") == nil {
		t.Fatal("nil database accepted")
	}
	for _, cmd := range []string{"up", "down", "status", "invalid"} {
		t.Run(cmd, func(t *testing.T) {
			db, m, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if Run(ctx, db, cmd) == nil {
				t.Fatal("expected error")
			}
			if err = m.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
