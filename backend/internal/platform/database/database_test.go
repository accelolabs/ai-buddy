package database

import (
	"context"
	"testing"
)

func TestOpenFailure(t *testing.T) {
	for _, dsn := range []string{"not-a-dsn", "postgres://user:pass@127.0.0.1:1/test?sslmode=disable"} {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		db, err := Open(ctx, dsn)
		if err == nil || db != nil {
			t.Fatal("expected connection failure")
		}
	}
}
