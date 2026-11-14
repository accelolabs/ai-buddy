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

	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/platform/database"
	"ai-buddy/internal/platform/server"
	"ai-buddy/internal/tracker/config"
	trackerhttp "ai-buddy/internal/tracker/http"
	"ai-buddy/internal/tracker/postgres"
	"ai-buddy/internal/tracker/progress"
	"ai-buddy/internal/tracker/tasks"
)

var openDatabase = database.Open

func serve(ctx context.Context, c config.Config) error {
	return server.Run(ctx, c.Config, openDatabase, func(db *sql.DB, login *auth.Auth) (http.Handler, error) {
		loc, err := time.LoadLocation(c.AppTimezone)
		if err != nil {
			return nil, errors.New("invalid APP_TIMEZONE")
		}

		repository := postgres.New(db)
		taskService := tasks.New(repository, repository, loc, time.Now)
		progressService := progress.New(repository, repository, loc, time.Now)
		return trackerhttp.Routes(
			login,
			trackerhttp.New(taskService, repository, progressService),
			c.BuddyAPIKey,
			c.TicketAPIKey,
		), nil
	})
}

func run() error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return serve(ctx, c)
}

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
