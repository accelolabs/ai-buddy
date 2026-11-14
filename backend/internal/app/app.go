package app

import (
	"ai-buddy/internal/buddy"
	"ai-buddy/internal/buddy/agent"
	buddyhttp "ai-buddy/internal/buddy/http"
	"ai-buddy/internal/buddy/mcp"
	buddyollama "ai-buddy/internal/buddy/ollama"
	buddypostgres "ai-buddy/internal/buddy/postgres"
	"ai-buddy/internal/config"
	"ai-buddy/internal/local"
	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/ticket/tickets"
	tickethttp "ai-buddy/internal/ticket/tickets/http"
	ticketpostgres "ai-buddy/internal/ticket/tickets/postgres"
	trackerhttp "ai-buddy/internal/tracker/http"
	trackerpostgres "ai-buddy/internal/tracker/postgres"
	"ai-buddy/internal/tracker/progress"
	"ai-buddy/internal/tracker/tasks"
	"ai-buddy/internal/users"
	"ai-buddy/internal/wiki"
	wikihttp "ai-buddy/internal/wiki/http"
	wikiollama "ai-buddy/internal/wiki/ollama"
	wikipostgres "ai-buddy/internal/wiki/postgres"
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
	profiles := &users.Repository{DB: db}
	selector := &users.Selector{Profiles: profiles}
	trackerRepo := trackerpostgres.New(db)
	taskService := tasks.New(trackerRepo, trackerRepo, loc, time.Now)
	progressService := progress.New(trackerRepo, trackerRepo, loc, time.Now)
	ticketService := tickets.New(ticketpostgres.New(db), local.Mentors{Repository: trackerRepo})
	wikiService := &wiki.Service{Store: wikipostgres.New(db), Embedder: wikiollama.New(c.Ollama.URL), Minimum: c.Wiki.MinSimilarity}
	repo := buddypostgres.New(db)
	if err := repo.Recover(ctx); err != nil {
		return nil, err
	}
	tools := &mcp.Tools{Repo: repo, Tracker: local.Tracker{Tasks: taskService, ProgressService: progressService}, Ticket: local.Ticket{Service: ticketService}, Wiki: local.Wiki{Service: wikiService, Profiles: profiles}}
	worker, stop := context.WithCancel(ctx)
	service := &buddy.Service{Repo: repo, Runner: agent.New(repo, tools, buddyollama.New(c.Ollama.URL)), Executor: tools, Context: worker}
	mux := http.NewServeMux()
	selector.Register(mux)
	trackerhttp.Register(mux, selector, trackerhttp.New(taskService, trackerRepo, progressService))
	tickethttp.Register(mux, selector, tickethttp.New(ticketService))
	wikihttp.Register(mux, selector, &wikihttp.Handler{Service: wikiService})
	buddyhttp.New(service).Register(mux, selector)
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if db.PingContext(ctx) != nil {
			httpx.JSON(w, 503, map[string]string{"error": "database unavailable"})
			return
		}
		w.WriteHeader(204)
	})
	httpx.Fallbacks(mux, map[string][]string{
		"/api/users": {"GET"}, "/api/me": {"GET"}, "/api/tasks": {"GET", "POST"}, "/api/tasks/{taskID}/complete": {"POST"}, "/api/progress": {"GET"}, "/api/interns": {"GET"}, "/api/interns/{id}/progress": {"GET"}, "/api/tickets": {"GET", "POST"}, "/api/tickets/incoming": {"GET"}, "/api/search": {"POST"}, "/api/documents": {"GET"}, "/api/documents/{id}": {"GET"}, "/api/chats": {"GET", "POST"}, "/api/chats/{id}": {"GET"}, "/api/chats/{id}/messages": {"POST"}, "/api/runs/{id}/events": {"GET"}, "/api/runs/{id}/retry": {"POST"}, "/api/actions/{id}/confirm": {"POST"}, "/api/actions/{id}/cancel": {"POST"},
	})
	return &App{Handler: mux, Buddy: service, stop: stop}, nil
}
func (a *App) Close() { a.stop(); a.Buddy.Close() }
