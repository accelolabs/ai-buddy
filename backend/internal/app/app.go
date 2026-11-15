// Package app wires the demo domains and manages application shutdown.
package app

import (
	"ai-buddy/internal/buddy"
	"ai-buddy/internal/config"
	"ai-buddy/internal/ollama"
	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/ticket"
	"ai-buddy/internal/tracker"
	"ai-buddy/internal/users"
	"ai-buddy/internal/wiki"
	"context"
	"database/sql"
	"net/http"
	"time"
)

type App struct {
	Handler http.Handler
	Buddy   *buddy.Service
	stop    context.CancelFunc
}

func New(ctx context.Context, db *sql.DB, c config.Config) (*App, error) {
	loc, err := time.LoadLocation(c.Tracker.Timezone)
	if err != nil {
		return nil, err
	}

	model := ollama.New(c.Ollama.URL)
	profiles := &users.Store{DB: db}
	selector := &users.Selector{Profiles: profiles}
	trackerRepo := &tracker.Store{DB: db}
	taskService := &tracker.Service{
		Store:    trackerRepo,
		Profiles: profiles,
		Location: loc,
		Now:      time.Now,
	}

	ticketService := ticket.New(&ticket.Store{DB: db}, profiles)
	wikiService := &wiki.Service{
		Store:    &wiki.Store{DB: db},
		Embedder: model,
		Minimum:  c.Wiki.MinSimilarity,
		Profiles: profiles,
	}

	repo := &buddy.Store{DB: db}
	if err := repo.Recover(ctx); err != nil {
		return nil, err
	}

	tools := &buddy.Tools{
		Repo:    repo,
		Tracker: taskService,
		Ticket:  ticketService,
		Wiki:    wikiService,
	}
	worker, stop := context.WithCancel(ctx)
	service := &buddy.Service{
		Repo:     repo,
		Runner:   buddy.NewAgent(repo, tools, model),
		Executor: tools,
		Context:  worker,
	}
	mux := http.NewServeMux()
	selector.Register(mux)
	tracker.Register(mux, selector, taskService)
	ticket.Register(mux, selector, ticketService)
	wiki.Register(mux, selector, wikiService)
	buddy.NewHandler(service).Register(mux, selector)
	mux.HandleFunc(
		"GET /health/live",
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
	)
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if db.PingContext(ctx) != nil {
			httpx.JSON(w, 503, map[string]string{"error": "database unavailable"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
	httpx.Fallbacks(mux, map[string][]string{
		"/api/users": {
			http.MethodGet,
		},
		"/api/me":                      {http.MethodGet},
		"/api/tasks":                   {http.MethodPost},
		"/api/tasks/{taskID}/complete": {http.MethodPost},
		"/api/progress":                {http.MethodGet},
		"/api/interns":                 {http.MethodGet},
		"/api/interns/{id}/progress":   {http.MethodGet},
		"/api/tickets":                 {http.MethodGet, http.MethodPost},
		"/api/tickets/incoming":        {http.MethodGet},
		"/api/documents":               {http.MethodGet},
		"/api/documents/{id}":          {http.MethodGet},
		"/api/chats":                   {http.MethodGet, http.MethodPost},
		"/api/chats/{id}":              {http.MethodGet},
		"/api/chats/{id}/messages":     {http.MethodPost},
		"/api/runs/{id}/events":        {http.MethodGet},
		"/api/runs/{id}/retry":         {http.MethodPost},
		"/api/actions/{id}/confirm":    {http.MethodPost},
		"/api/actions/{id}/cancel":     {http.MethodPost},
	})

	return &App{Handler: mux, Buddy: service, stop: stop}, nil
}
func (a *App) Close() { a.stop(); a.Buddy.Close() }
