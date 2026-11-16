package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Project packages.
	"backend/internal/buddy/agent"
	buddyhandler "backend/internal/buddy/handler"
	"backend/internal/config"
	"backend/internal/ollama"
	tickethandler "backend/internal/ticket/handler"
	ticketrepository "backend/internal/ticket/repository"
	ticketservice "backend/internal/ticket/service"
	trackerhandler "backend/internal/tracker/handler"
	trackerrepository "backend/internal/tracker/repository"
	trackerservice "backend/internal/tracker/service"
	usershandler "backend/internal/users/handler"
	"backend/internal/users/middleware"
	usersrepository "backend/internal/users/repository"
	usersservice "backend/internal/users/service"
	wikihandler "backend/internal/wiki/handler"
	"backend/internal/wiki/rag"
	wikirepository "backend/internal/wiki/repository"
	wikiservice "backend/internal/wiki/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type dependencies struct {
	load   func() (config.Config, error)
	open   func(string, string) (*sql.DB, error)
	routes func(*sql.DB, config.Config) (http.Handler, error)
	serve  func(context.Context, *http.Server) error
}

func defaultDependencies() dependencies {
	return dependencies{
		load:   config.Load,
		open:   sql.Open,
		routes: routes,
		serve:  serve,
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, defaultDependencies()); err != nil {
		log.Print(err)
		os.Exit(1)
	}
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
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = db.PingContext(pingCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	handler, err := deps.routes(db, cfg)
	if err != nil {
		return err
	}

	return deps.serve(ctx, &http.Server{Addr: cfg.Addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second})
}

func routes(db *sql.DB, cfg config.Config) (http.Handler, error) {
	userRepository := usersrepository.NewRepository(db)
	userService := usersservice.NewService(userRepository)
	userHandler := usershandler.NewHandler(userService)
	wikiRepository := wikirepository.NewRepository(db)
	wikiHandler := wikihandler.NewHandler(wikiservice.NewService(wikiRepository, cfg.KnowledgeBaseDir))
	search := rag.NewRAG(wikiRepository, ollama.NewClient(cfg.OllamaURL, cfg.EmbeddingModel), cfg.RAGChunkCount)
	trackerService := trackerservice.NewService(trackerrepository.NewRepository(db))
	buddyAgent, err := agent.NewAgent(
		ollama.NewChatClient(cfg.OllamaURL, cfg.ChatModel, cfg.ChatContextLength),
		search,
		cfg.SystemPromptFile,
		userService,
		trackerService,
	)
	if err != nil {
		return nil, err
	}

	buddyHandler := buddyhandler.NewHandler(buddyAgent, cfg.ChatHistoryLimit)
	ticketHandler := tickethandler.NewHandler(ticketservice.NewService(ticketrepository.NewRepository(db)))
	trackerHandler := trackerhandler.NewHandler(trackerService)
	selectedUser := middleware.Middleware(userService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/users", userHandler.List)
	mux.Handle("GET /api/users/interns", selectedUser(http.HandlerFunc(userHandler.ListInterns)))

	mux.Handle("GET /api/tracker/tasks", selectedUser(http.HandlerFunc(trackerHandler.List)))
	mux.Handle(
		"GET /api/tracker/interns/{internID}/tasks",
		selectedUser(http.HandlerFunc(trackerHandler.ListIntern)),
	)
	mux.Handle("POST /api/tracker/tasks", selectedUser(http.HandlerFunc(trackerHandler.Create)))
	mux.Handle(
		"POST /api/tracker/tasks/{id}/complete",
		selectedUser(http.HandlerFunc(trackerHandler.Complete)),
	)

	mux.Handle("POST /api/buddy/messages", selectedUser(http.HandlerFunc(buddyHandler.Send)))

	mux.Handle("POST /api/tickets", selectedUser(http.HandlerFunc(ticketHandler.Create)))
	mux.Handle("GET /api/tickets", selectedUser(http.HandlerFunc(ticketHandler.List)))
	mux.Handle("GET /api/tickets/{id}", selectedUser(http.HandlerFunc(ticketHandler.Get)))

	mux.Handle("GET /api/wiki", selectedUser(http.HandlerFunc(wikiHandler.List)))
	mux.Handle("GET /api/wiki/{id}", selectedUser(http.HandlerFunc(wikiHandler.Get)))

	return mux, nil
}

func serve(ctx context.Context, server *http.Server) error {
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()
	log.Printf("HTTP server starting on %s", server.Addr)
	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}

		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			<-serverErr
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}

		err := <-serverErr
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}

		return nil
	}
}
