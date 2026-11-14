package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"ai-buddy/backend-tracker/internal/config"
	"ai-buddy/backend-tracker/internal/database"
	"ai-buddy/backend-tracker/internal/seed"
)

var openDatabase = database.Open

func run() error {
	c, err := config.LoadDatabase()
	if err != nil {
		return err
	}

	seedConfig, err := seed.LoadConfig()
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

	return seed.Run(ctx, db, seedConfig)
}

func main() {
	if run() != nil {
		slog.Error("seed failed; check database and Authentik seed configuration")
		os.Exit(1)
	}
}
