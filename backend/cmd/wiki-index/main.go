package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-buddy/internal/platform/database"
	"ai-buddy/internal/wiki"
	"ai-buddy/internal/wiki/indexer"
	"ai-buddy/internal/wiki/ollama"
	"ai-buddy/internal/wiki/postgres"
)

var openDatabase = database.Open

func run(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("wiki-index", flag.ContinueOnError)
	flags.SetOutput(out)
	full := flags.Bool("full", false, "recreate all embeddings")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: wiki-index [--full]")
	}
	c, err := wiki.LoadIndexConfig()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	db, err := openDatabase(ctx, c.URL)
	if err != nil {
		return err
	}
	defer db.Close()
	stats, err := indexer.Run(ctx, postgres.New(db), c.Root, ollama.New(c.OllamaURL), *full)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "added=%d changed=%d deleted=%d skipped=%d\n", stats.Added, stats.Changed, stats.Deleted, stats.Skipped)
	return err
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "wiki-index:", err)
		os.Exit(1)
	}
}
