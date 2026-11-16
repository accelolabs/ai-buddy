package main

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	// Project packages.
	"backend/internal/config"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRun(t *testing.T) {
	failure := errors.New("dependency failed")

	for _, mode := range []string{"ok", "config", "open", "cancel", "users", "tickets", "tasks"} {
		t.Run(mode, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			if err != nil {
				t.Fatal(err)
			}

			defer func() { _ = db.Close() }()
			calls := []string{}
			cfg := config.Config{DatabaseURL: "mock-url", Addr: "localhost:8080"}
			deps := defaultDependencies()
			deps.load = func() (config.Config, error) {
				calls = append(calls, "config")
				if mode == "config" {
					return config.Config{}, failure
				}

				return cfg, nil
			}

			deps.open = func(driver, dsn string) (*sql.DB, error) {
				calls = append(calls, "open")
				if driver != "pgx" || dsn != cfg.DatabaseURL {
					t.Fatalf("open %s %s", driver, dsn)
				}

				if mode == "open" {
					return nil, failure
				}

				return db, nil
			}

			for _, stage := range []string{"users", "tickets", "tasks"} {
				f := func(ctx context.Context, gotDB *sql.DB) error {
					calls = append(calls, stage)
					if gotDB != db || ctx.Err() != nil {
						t.Fatal("seed dependencies")
					}

					if mode == stage {
						return failure
					}

					return nil
				}

				switch stage {
				case "users":
					deps.users = f
				case "tickets":
					deps.tickets = f
				case "tasks":
					deps.tasks = f
				}
			}

			if mode != "config" && mode != "open" {
				mock.ExpectClose()
			}

			ctx := t.Context()
			if mode == "cancel" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			if mode == "ok" {
				err = execute(deps)
			} else {
				err = run(ctx, deps)
			}

			switch mode {
			case "ok":
				if err != nil {
					t.Fatal(err)
				}
			case "cancel":
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			default:
				if !errors.Is(err, failure) {
					t.Fatal(err)
				}
			}

			expected := []string{"config"}
			if mode != "config" {
				expected = append(expected, "open")
				if mode != "open" && mode != "cancel" {
					for _, stage := range []string{"users", "tickets", "tasks"} {
						expected = append(expected, stage)
						if mode == stage {
							break
						}
					}
				}
			}

			if !reflect.DeepEqual(calls, expected) {
				t.Fatalf("calls %v want %v", calls, expected)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
