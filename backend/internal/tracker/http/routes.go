package trackerhttp

import (
	"net/http"
	"strings"

	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/platform/httpx"
)

func Routes(
	login *auth.Auth,
	h *Handler,
	buddyKey, ticketKey string,
) http.Handler {
	mux := http.NewServeMux()
	browser := http.NewServeMux()
	browserMethods := map[string][]string{}
	internalMethods := map[string][]string{}
	internal := http.NewServeMux()
	for _, route := range []struct {
		pattern, role string
		handler       http.HandlerFunc
	}{
		{"GET /api/tasks", "intern", h.ListTasks},
		{"POST /api/tasks", "intern", h.Create},
		{"POST /api/tasks/{taskID}/complete", "intern", h.Complete},
		{"GET /api/progress", "intern", h.Progress},
		{"GET /api/interns", "mentor", h.ListInterns},
		{"GET /api/interns/{id}/progress", "mentor", h.MentorProgress},
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
		{"GET /internal/interns/{id}/progress", false, h.Progress},
		{"POST /internal/interns/{id}/tasks", false, h.Create},
		{"POST /internal/interns/{id}/tasks/{taskID}/complete", false, h.Complete},
		{"GET /internal/interns/{id}/mentor", true, h.Mentor},
	} {
		internal.Handle(route.pattern, httpx.KeyAuth(route.handler, buddyKey, ticketKey, route.ticket))
		method, path, _ := strings.Cut(route.pattern, " ")
		internalMethods[path] = append(internalMethods[path], method)
	}

	httpx.Fallbacks(internal, internalMethods)
	mux.Handle("/internal/", internal)
	return mux
}
