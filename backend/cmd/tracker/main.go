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
	"ai-buddy/internal/tracker/interns"
	internshttp "ai-buddy/internal/tracker/interns/http"
	internspostgres "ai-buddy/internal/tracker/interns/postgres"
	"ai-buddy/internal/tracker/progress"
	progresshttp "ai-buddy/internal/tracker/progress/http"
	"ai-buddy/internal/tracker/tasks"
	taskshttp "ai-buddy/internal/tracker/tasks/http"
	taskspostgres "ai-buddy/internal/tracker/tasks/postgres"
)

var openDatabase = database.Open

func serve(ctx context.Context, c config.Config) error {
	return server.Run(ctx, c.Config, openDatabase, func(db *sql.DB, login *auth.Auth) (http.Handler, error) {
		loc, err := time.LoadLocation(c.AppTimezone)
		if err != nil {
			return nil, errors.New("invalid APP_TIMEZONE")
		}

		internRepository := internspostgres.New(db)
		taskRepository := taskspostgres.New(db)
		internService := interns.New(internRepository)
		taskService := tasks.New(taskRepository, internService, loc, time.Now)
		progressService := progress.New(internRepository, taskRepository, loc, time.Now)
		api := domainRoutes(
			login,
			taskshttp.New(taskService),
			internshttp.New(internService),
			progresshttp.New(progressService),
			c.BuddyAPIKey,
			c.TicketAPIKey,
		)
		mux := http.NewServeMux()
		mux.Handle("/api/", api)
		mux.Handle("/internal/", api)
		return mux, nil
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
