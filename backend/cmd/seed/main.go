package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	"ai-buddy/internal/platform/cli"
	ticket "ai-buddy/internal/ticket/seed"
	tracker "ai-buddy/internal/tracker/seed"
)

var openDatabase = cli.Open

func prepare(service, command string) (cli.Operation, error) {
	if service == "tracker" {
		c, err := tracker.LoadConfig()
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, db *sql.DB) error { return tracker.Run(ctx, db, c) }, nil
	}
	c, err := ticket.LoadConfig()
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, db *sql.DB) error { return ticket.Run(ctx, db, c) }, nil
}
func run() error { return cli.Run(os.Args[1:], false, prepare, openDatabase) }
func main() {
	if run() != nil {
		slog.Error("seed failed; check service, command and configuration")
		os.Exit(1)
	}
}
