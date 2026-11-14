package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-buddy/backend-tracker/internal/auth"
	"ai-buddy/backend-tracker/internal/config"
	"ai-buddy/backend-tracker/internal/database"
	"ai-buddy/backend-tracker/internal/tracker"
)

var openDatabase = database.Open

func serve(ctx context.Context, c config.Config) error {
	startup, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	db, err := openDatabase(startup, c.URL)
	if err != nil {
		return errors.New("database connection failed")
	}
	defer db.Close()
	sessions, stop := auth.NewSessions(db, c)
	defer stop()
	provider, err := auth.NewOIDCProvider(startup, c)
	if err != nil {
		return err
	}
	login := auth.New(sessions, provider)
	loc, err := time.LoadLocation(c.AppTimezone)
	if err != nil {
		return errors.New("invalid APP_TIMEZONE")
	}
	api := (&tracker.API{Service: tracker.New(&tracker.Postgres{DB: db}, loc), Auth: login, BuddyKey: c.BuddyAPIKey, TicketKey: c.TicketAPIKey}).Handler()
	return serveHTTP(ctx, c.HTTPAddr, routes(db, login.Handler(), api))
}

func routes(db *sql.DB, login http.Handler, api ...http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/auth/", login)
	if len(api) > 0 {
		mux.Handle("/api/", api[0])
		mux.Handle("/internal/", api[0])
	}
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if db.PingContext(ctx) != nil {
			http.Error(w, "database unavailable", 503)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

func serveHTTP(ctx context.Context, addr string, handler http.Handler) error {
	server := &http.Server{
		Addr: addr, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       60 * time.Second,
		WriteTimeout:      0, // Future SSE responses may outlive a fixed write timeout.
	}
	defer server.Close()
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	slog.Info("server listening", "address", addr)
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
func main() {
	c := config.MustLoad()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := serve(ctx, c); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
