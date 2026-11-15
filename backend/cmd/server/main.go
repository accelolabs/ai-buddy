package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-buddy/internal/app"
	"ai-buddy/internal/config"
	"ai-buddy/internal/migrations"
	"ai-buddy/internal/ollama"
	"ai-buddy/internal/platform/cleanup"
	"ai-buddy/internal/platform/database"
	"ai-buddy/internal/seed"
	"ai-buddy/internal/wiki"
)

func run(ctx context.Context, args []string) error {
	command := "serve"
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}
	full := false
	switch command {
	case "serve", "seed":
		if len(args) != 0 {
			return errors.New("unexpected arguments")
		}
	case "migrate":
		if len(args) != 1 || (args[0] != "up" && args[0] != "down" && args[0] != "status") {
			return errors.New("usage: server migrate up|down|status")
		}
	case "index":
		flags := flag.NewFlagSet("index", flag.ContinueOnError)
		flags.BoolVar(&full, "full", false, "recreate all embeddings")
		if err := flags.Parse(args); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("usage: server index [--full]")
		}
	default:
		return errors.New("usage: server [serve|migrate up|down|status|seed|index [--full]]")
	}
	c, err := config.Load()
	if err != nil {
		return err
	}
	startup, cancel := context.WithTimeout(ctx, 30*time.Second)
	db, err := database.Open(startup, c.Database.URL)
	cancel()
	if err != nil {
		return err
	}
	defer cleanup.Close(db)
	if command == "migrate" {
		return migrations.Run(ctx, db, args[0])
	}
	if command == "index" {
		stats, err := wiki.Run(ctx, &wiki.Store{DB: db}, c.Wiki.KnowledgeBaseDir, ollama.New(c.Ollama.URL), full)
		if err == nil {
			fmt.Printf("added=%d changed=%d deleted=%d skipped=%d\n", stats.Added, stats.Changed, stats.Deleted, stats.Skipped)
		}
		return err
	}
	if err = migrations.Run(ctx, db, "up"); err != nil {
		return err
	}
	if err = seed.Run(ctx, db); err != nil {
		return err
	}
	if command == "seed" {
		return nil
	}
	application, err := app.New(ctx, db, c)
	if err != nil {
		return err
	}
	defer application.Close()
	server := &http.Server{Addr: c.HTTP.Addr, Handler: application.Handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	defer cleanup.Close(server)
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	select {
	case err = <-done:
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
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
