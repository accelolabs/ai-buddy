package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/platform/database"
	"ai-buddy/internal/platform/server"
	"ai-buddy/internal/ticket/config"
	"ai-buddy/internal/ticket/tickets"
	ticketshttp "ai-buddy/internal/ticket/tickets/http"
	"ai-buddy/internal/ticket/tickets/postgres"
	"ai-buddy/internal/ticket/tracker"
)

var openDatabase = database.Open

func serve(ctx context.Context, c config.Config) error {
	return server.Run(ctx, c.Config, openDatabase, func(db *sql.DB, login *auth.Auth) (http.Handler, error) {
		trackerClient := tracker.New(
			c.TrackerURL,
			c.TicketAPIKey,
			c.TrackerTimeout,
		)
		service := tickets.New(postgres.New(db), trackerClient)
		handler := domainRoutes(
			login,
			ticketshttp.New(service),
			c.BuddyAPIKey,
		)

		return handler, nil
	})
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
