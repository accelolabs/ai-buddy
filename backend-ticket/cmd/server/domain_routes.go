package main

import (
	"net/http"

	"ai-buddy/backend-ticket/internal/auth"
	"ai-buddy/backend-ticket/internal/httpx"
	ticketshttp "ai-buddy/backend-ticket/internal/tickets/http"
)

func domainRoutes(login *auth.Auth, handler *ticketshttp.Handler, key string) http.Handler {
	mux := http.NewServeMux()
	browser := http.NewServeMux()
	browser.Handle("GET /api/tickets", login.RequireRoles(http.HandlerFunc(handler.Sent), auth.RoleIntern))
	browser.Handle("POST /api/tickets", login.RequireRoles(http.HandlerFunc(handler.Create), auth.RoleIntern))
	browser.Handle("GET /api/tickets/incoming", login.RequireRoles(http.HandlerFunc(handler.Incoming), auth.RoleMentor))
	httpx.Fallbacks(browser, map[string][]string{
		"/api/tickets":          {"GET", "POST"},
		"/api/tickets/incoming": {"GET"},
	})
	mux.Handle("/api/", login.WithSession(browser))
	internal := http.NewServeMux()
	internal.HandleFunc("GET /internal/interns/{id}/tickets", handler.Sent)
	internal.HandleFunc("POST /internal/interns/{id}/tickets", handler.Create)
	httpx.Fallbacks(internal, map[string][]string{
		"/internal/interns/{id}/tickets": {"GET", "POST"},
	})
	mux.Handle("/internal/", httpx.KeyAuth(internal, key, "", false))
	httpx.Fallbacks(mux, nil)
	return mux
}
