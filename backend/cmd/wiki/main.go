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
	"ai-buddy/internal/wiki"
	"ai-buddy/internal/wiki/authentik"
	wikihttp "ai-buddy/internal/wiki/http"
	"ai-buddy/internal/wiki/ollama"
	"ai-buddy/internal/wiki/postgres"
)

var openDatabase = database.Open

func serve(ctx context.Context, c wiki.Config) error {
	return server.Run(ctx, c.Config, openDatabase, func(db *sql.DB, login *auth.Auth) (http.Handler, error) {
		service := &wiki.Service{
			Store:    postgres.New(db),
			Embedder: ollama.New(c.OllamaURL),
			Minimum:  c.Minimum,
		}
		h := &wikihttp.Handler{
			Service: service,
			Teams:   authentik.New(c.AuthentikURL, c.AuthentikToken),
		}
		return wikihttp.Routes(login, h, c.BuddyAPIKey), nil
	})
}

func run() error {
	c, err := wiki.LoadConfig()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return serve(ctx, c)
}

func main() {
	if err := run(); err != nil {
		slog.Error("wiki stopped", "error", err)
		os.Exit(1)
	}
}
