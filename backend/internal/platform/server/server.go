package server

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/platform/cleanup"
	"ai-buddy/internal/platform/config"
)

type OpenDatabase func(context.Context, string) (*sql.DB, error)
type Domain func(*sql.DB, *auth.Auth) (http.Handler, error)

func Run(ctx context.Context, c config.Config, open OpenDatabase, domain Domain) error {
	startup, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	db, err := open(startup, c.URL)
	if err != nil {
		return errors.New("database connection failed")
	}
	defer cleanup.Close(db)
	sessions, stop := auth.NewSessions(db, c)
	defer stop()
	provider, err := auth.NewOIDCProvider(startup, c)
	if err != nil {
		return err
	}
	login := auth.New(sessions, provider)
	api, err := domain(db, login)
	if err != nil {
		return err
	}
	if closer, ok := api.(io.Closer); ok {
		defer cleanup.Close(closer)
	}
	return ServeHTTP(ctx, c.HTTPAddr, Routes(db, login.Handler(), api))
}

func Routes(db *sql.DB, login, domain http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/auth/", login)
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if db.PingContext(ctx) != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("/", domain)
	return mux
}

func ServeHTTP(ctx context.Context, addr string, handler http.Handler) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       60 * time.Second,
		WriteTimeout:      0, // Future SSE responses may outlive a fixed write timeout.
	}
	defer cleanup.Close(server)
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	slog.Info("server listening", "address", addr)
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
