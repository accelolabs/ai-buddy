package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	buddy "ai-buddy/internal/buddy/migrations"
	"ai-buddy/internal/platform/cli"
	"ai-buddy/internal/platform/migrations"
	ticket "ai-buddy/internal/ticket/migrations"
	tracker "ai-buddy/internal/tracker/migrations"
	wiki "ai-buddy/internal/wiki/migrations"
)

var openDatabase = cli.Open

func prepare(service, command string) (cli.Operation, error) {
	files := tracker.Files
	if service == "buddy" {
		files = buddy.Files
	}
	if service == "ticket" {
		files = ticket.Files
	}
	if service == "wiki" {
		files = wiki.Files
	}
	return func(ctx context.Context, db *sql.DB) error { return migrations.Run(ctx, db, files, command) }, nil
}
func run() error { return cli.Run(os.Args[1:], true, prepare, openDatabase) }
func main() {
	if run() != nil {
		slog.Error("migrate failed; check service, command and configuration")
		os.Exit(1)
	}
}
