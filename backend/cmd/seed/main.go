package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	// Project packages.
	"backend/internal/config"
	"backend/internal/ticket"
	"backend/internal/tracker"
	"backend/internal/users"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	defer func() {
		_ = db.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := users.Seed(ctx, db); err != nil {
		log.Fatal(err)
	}

	if err := ticket.Seed(ctx, db); err != nil {
		log.Fatal(err)
	}

	if err := tracker.Seed(ctx, db); err != nil {
		log.Fatal(err)
	}

	log.Print("demo users, tickets and tasks seeded")
}
