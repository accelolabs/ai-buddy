package main

import (
	"net/http"
	"strings"

	"ai-buddy/backend-tracker/internal/auth"
	"ai-buddy/backend-tracker/internal/httpx"
	internshttp "ai-buddy/backend-tracker/internal/interns/http"
	progresshttp "ai-buddy/backend-tracker/internal/progress/http"
	taskshttp "ai-buddy/backend-tracker/internal/tasks/http"
)

func domainRoutes(
	login *auth.Auth,
	taskHandler *taskshttp.Handler,
	internHandler *internshttp.Handler,
	progressHandler *progresshttp.Handler,
	buddyKey, ticketKey string,
) http.Handler {
	mux := http.NewServeMux()
	browser := http.NewServeMux()
	browserMethods := map[string][]string{}
	internalMethods := map[string][]string{}
	for _, route := range []struct {
		pattern, role string
		handler       http.HandlerFunc
	}{
		{"GET /api/tasks", "intern", taskHandler.List},
		{"POST /api/tasks", "intern", taskHandler.Create},
		{"POST /api/tasks/{taskID}/complete", "intern", taskHandler.Complete},
		{"GET /api/progress", "intern", progressHandler.Get},
		{"GET /api/interns", "mentor", internHandler.List},
		{"GET /api/interns/{id}/progress", "mentor", progressHandler.Mentor},
	} {
		browser.Handle(route.pattern, login.RequireRoles(route.handler, route.role))
		method, path, _ := strings.Cut(route.pattern, " ")
		browserMethods[path] = append(browserMethods[path], method)
	}

	httpx.Fallbacks(browser, browserMethods)
	mux.Handle("/api/", login.WithSession(browser))
	for _, route := range []struct {
		pattern string
		ticket  bool
		handler http.HandlerFunc
	}{
		{"GET /internal/interns/{id}/progress", false, progressHandler.Get},
		{"POST /internal/interns/{id}/tasks", false, taskHandler.Create},
		{"POST /internal/interns/{id}/tasks/{taskID}/complete", false, taskHandler.Complete},
		{"GET /internal/interns/{id}/mentor", true, internHandler.Mentor},
	} {
		mux.Handle(route.pattern, httpx.KeyAuth(route.handler, buddyKey, ticketKey, route.ticket))
		method, path, _ := strings.Cut(route.pattern, " ")
		internalMethods[path] = append(internalMethods[path], method)
	}

	httpx.Fallbacks(mux, internalMethods)
	return mux
}
