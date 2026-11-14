package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"ai-buddy/backend-template/internal/config"
	"ai-buddy/backend-template/internal/database"
	"ai-buddy/backend-template/internal/seed"
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
	return seed.Run(ctx, db)
}
func main() {
	if run() != nil {
		slog.Error("seed failed; check database configuration")
		os.Exit(1)
	}
}
