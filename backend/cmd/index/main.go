package main

import (
	"backend/internal/config"
	"backend/internal/ollama"
	"backend/internal/wiki/indexer"
	wikirepository "backend/internal/wiki/repository"
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	defer func() { _ = db.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	wikiIndexer := indexer.NewIndexer(
		wikirepository.NewRepository(db),
		ollama.NewClient(cfg.OllamaURL, cfg.EmbeddingModel),
		cfg.KnowledgeBaseDir,
	)
	if err := wikiIndexer.Index(ctx); err != nil {
		log.Fatal(err)
	}
	log.Print("knowledge base indexed")
}
