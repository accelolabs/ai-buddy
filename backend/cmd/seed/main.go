package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	// Project packages.
	"backend/internal/config"
	"backend/internal/ticket"
	"backend/internal/tracker"
	"backend/internal/users"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type dependencies struct {
	load    func() (config.Config, error)
	open    func(string, string) (*sql.DB, error)
	users   func(context.Context, *sql.DB) error
	tickets func(context.Context, *sql.DB) error
	tasks   func(context.Context, *sql.DB) error
}

func main() {
	if err := execute(defaultDependencies()); err != nil {
		log.Fatal(err)
	}
}

func execute(deps dependencies) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	return run(ctx, deps)
}

func run(ctx context.Context, deps dependencies) error {
	cfg, err := deps.load()
	if err != nil {
		return err
	}

	db, err := deps.open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	defer func() {
		_ = db.Close()
	}()
	if err := ctx.Err(); err != nil {
		return err
	}

	for _, stage := range []func(context.Context, *sql.DB) error{deps.users, deps.tickets, deps.tasks} {
		if err := stage(ctx, db); err != nil {
			return err
		}
	}

	log.Print("demo users, tickets and tasks seeded")

	return nil
}

func defaultDependencies() dependencies {
	return dependencies{
		load:  config.Load,
		open:  sql.Open,
		users: users.Seed, tickets: ticket.Seed, tasks: tracker.Seed,
	}
}
