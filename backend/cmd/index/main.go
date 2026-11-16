package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	// Project packages.
	"backend/internal/config"
	"backend/internal/ollama"
	"backend/internal/wiki/indexer"
	wikirepository "backend/internal/wiki/repository"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type dependencies struct {
	load  func() (config.Config, error)
	open  func(string, string) (*sql.DB, error)
	index func(context.Context, *sql.DB, config.Config) error
}

func main() {
	if err := execute(defaultDependencies()); err != nil {
		log.Fatal(err)
	}
}

func execute(deps dependencies) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

	if err := deps.index(ctx, db, cfg); err != nil {
		return err
	}

	log.Print("knowledge base indexed")

	return nil
}

func defaultDependencies() dependencies {
	return dependencies{
		load: config.Load,
		open: sql.Open,
		index: func(ctx context.Context, db *sql.DB, cfg config.Config) error {
			return indexer.NewIndexer(wikirepository.NewRepository(db),
				ollama.NewClient(cfg.OllamaURL, cfg.EmbeddingModel), cfg.KnowledgeBaseDir).Index(ctx)
		},
	}
}
