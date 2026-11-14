package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"ai-buddy/backend-ticket/internal/config"
	"ai-buddy/backend-ticket/internal/database"
	"ai-buddy/backend-ticket/internal/migrations"
)

var openDatabase = database.Open

func run() error {
	c, err := config.LoadDatabase()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, err := openDatabase(ctx, c.URL)
	if err != nil {
		return err
	}
	defer db.Close()
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	return migrations.Run(ctx, db, command)
}
func main() {
	if run() != nil {
		slog.Error("migration failed; check database configuration and migration command")
		os.Exit(1)
	}
}
