package main

import (
	"context"
	"database/sql"
	"errors"
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(ctx)
	cancel()
	if err != nil {
		log.Fatalf("ping database: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
		log.Fatal(err)
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

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	log.Printf("HTTP server starting on %s", cfg.Addr)
	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve HTTP: %v", err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("shutdown HTTP server: %v", err)
		}
	}
}
