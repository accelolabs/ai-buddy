package migrations

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCommandsAndErrors(t *testing.T) {
	if Run(context.Background(), nil, fstest.MapFS{}, "up") == nil {
		t.Fatal("nil database accepted")
	}

	for _, cmd := range []string{
		"up",
		"down",
		"status",
		"invalid",
	} {
		t.Run(cmd, func(t *testing.T) {
			db, m, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				// sqlmock cleanup after assertions; Close is not part of this scenario.
				_ = db.Close()
			}()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if Run(ctx, db, fstest.MapFS{"001_test.sql": &fstest.MapFile{Data: []byte("-- +goose Up\nSELECT 1;\n-- +goose Down\nSELECT 1;")}}, cmd) == nil {
				t.Fatal("expected error")
			}

			if err = m.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
