package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	// Project packages.
	"backend/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type dependencies struct {
	load    func() (config.Config, error)
	open    func(string, string) (*sql.DB, error)
	dialect func(string) error
	migrate func(context.Context, *sql.DB, string) error
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

	if err := deps.dialect("postgres"); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}

	if err := deps.migrate(ctx, db, "migrations"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

func defaultDependencies() dependencies {
	return dependencies{
		load:    config.Load,
		open:    sql.Open,
		dialect: goose.SetDialect,
		migrate: func(ctx context.Context, db *sql.DB, dir string) error {
			return goose.UpContext(ctx, db, dir)
		},
	}
}
