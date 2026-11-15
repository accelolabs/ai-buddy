package main

import (
	"backend/internal/config"
	usershandler "backend/internal/users/handler"
	"backend/internal/users/middleware"
	usersrepository "backend/internal/users/repository"
	wikihandler "backend/internal/wiki/handler"
	wikirepository "backend/internal/wiki/repository"
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	userHandler := usershandler.NewHandler(userRepository)
	wikiHandler := wikihandler.NewHandler(wikirepository.NewRepository(db), cfg.KnowledgeBaseDir)
	selectedUser := middleware.Middleware(userRepository)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/users", userHandler.List)
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
