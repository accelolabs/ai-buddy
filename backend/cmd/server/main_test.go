package main

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	// Project packages.
	"backend/internal/config"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRun(t *testing.T) {
	failure := errors.New("dependency failed")

	for _, mode := range []string{"ok", "config", "open", "cancel", "ping", "routes", "serve"} {
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

			deps.routes = func(gotDB *sql.DB, gotCfg config.Config) (http.Handler, error) {
				calls = append(calls, "routes")
				if gotDB != db || gotCfg != cfg {
					t.Fatal("route dependencies")
				}

				if mode == "routes" {
					return nil, failure
				}

				return http.NewServeMux(), nil
			}

			deps.serve = func(ctx context.Context, server *http.Server) error {
				calls = append(calls, "serve")
				if server.Addr != cfg.Addr || server.Handler == nil || server.ReadHeaderTimeout != 5*time.Second ||
					ctx.Err() != nil {
					t.Fatal("server dependencies")
				}

				if mode == "serve" {
					return failure
				}

				return nil
			}

			if mode != "config" && mode != "open" && mode != "cancel" {
				e := mock.ExpectPing()
				if mode == "ping" {
					e.WillReturnError(failure)
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

			err = run(ctx, deps)
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
					for _, stage := range []string{"routes", "serve"} {
						if mode == "ping" {
							break
						}

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

func TestRoutes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = db.Close() }()
	cfg := config.Config{SystemPromptFile: filepath.Join(t.TempDir(), "prompt"), ChatHistoryLimit: 6}
	if _, err := routes(db, cfg); err == nil {
		t.Fatal("missing prompt accepted")
	}

	if err := os.WriteFile(cfg.SystemPromptFile, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}

	handler, err := routes(db, cfg)
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"/api/users/interns", "/api/tracker/tasks", "/api/tracker/interns/intern/tasks",
		"/api/tickets", "/api/tickets/id", "/api/wiki", "/api/wiki/id",
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}

	for _, path := range []string{
		"/api/tracker/tasks", "/api/tracker/tasks/id/complete", "/api/buddy/messages", "/api/tickets",
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("POST", path, nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}

	mock.ExpectQuery("SELECT id, name, role, team FROM users ORDER BY id").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role", "team"}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/users", nil))
	if w.Code != 200 || w.Body.String() != "[]\n" {
		t.Fatalf("%d %s", w.Code, w.Body)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestServe(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = listener.Close() }()
	if err := serve(t.Context(), &http.Server{Addr: listener.Addr().String()}); err == nil {
		t.Fatal("listen error lost")
	}

	ctx, cancel := context.WithCancel(t.Context())
	started := make(chan struct{})
	server := &http.Server{
		Addr: "127.0.0.1:0", Handler: http.NewServeMux(),
		BaseContext: func(net.Listener) context.Context { close(started); return ctx },
	}

	done := make(chan error, 1)
	go func() { done <- serve(ctx, server) }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not start")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop")
	}
}
