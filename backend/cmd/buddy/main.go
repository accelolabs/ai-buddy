package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/buddy/agent"
	buddyhttp "ai-buddy/internal/buddy/http"
	"ai-buddy/internal/buddy/mcp"
	"ai-buddy/internal/buddy/ollama"
	"ai-buddy/internal/buddy/postgres"
	"ai-buddy/internal/buddy/remote"
	"ai-buddy/internal/buddy/ticket"
	"ai-buddy/internal/buddy/tracker"
	"ai-buddy/internal/buddy/wiki"
	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/platform/database"
	"ai-buddy/internal/platform/server"
)

var openDatabase = database.Open

func serve(ctx context.Context, c buddy.Config) error {
	worker, stop := context.WithCancel(ctx)
	defer stop()
	return server.Run(ctx, c.Config, openDatabase, func(db *sql.DB, login *auth.Auth) (http.Handler, error) {
		repo := postgres.New(db)
		startup, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := repo.Recover(startup); err != nil {
			return nil, err
		}
		tools := &mcp.Tools{Repo: repo, Wiki: &wiki.Client{Client: remote.New(c.WikiURL, c.BuddyAPIKey, 90*time.Second), PublicURL: c.WikiPublicURL}, Tracker: &tracker.Client{Client: remote.New(c.TrackerURL, c.BuddyAPIKey, 10*time.Second)}, Ticket: &ticket.Client{Client: remote.New(c.TicketURL, c.BuddyAPIKey, 10*time.Second)}}
		service := &buddy.Service{Repo: repo, Runner: agent.New(repo, tools, ollama.New(c.OllamaURL)), Executor: tools, Context: worker}
		return &ownedHandler{Handler: buddyhttp.New(service).Routes(login), stop: stop, service: service}, nil
	})
}

// Stop generators and wait for their final persistence before server.Run closes DB.
type ownedHandler struct {
	http.Handler
	stop    context.CancelFunc
	service *buddy.Service
}

func (h *ownedHandler) Close() error { h.stop(); h.service.Close(); return nil }

func run() error {
	c, err := buddy.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return serve(ctx, c)
}
func main() {
	if err := run(); err != nil {
		slog.Error("Buddy stopped", "error", err)
		os.Exit(1)
	}
}
