package tracker

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"ai-buddy/backend-tracker/internal/auth"
)

type API struct {
	Service             *Service
	Auth                *auth.Auth
	BuddyKey, TicketKey string
}

func jsonResponse(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
func failure(w http.ResponseWriter, e error) {
	code, message := 500, "storage unavailable"
	switch {
	case errors.Is(e, ErrInvalid):
		code, message = 400, "invalid input"
	case errors.Is(e, ErrNotFound):
		code, message = 404, "not found"
	case errors.Is(e, ErrConflict):
		code, message = 409, ErrConflict.Error()
	}
	jsonResponse(w, code, map[string]string{"error": message})
}
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	browser := http.NewServeMux()
	browserMethods := map[string][]string{}
	internalMethods := map[string][]string{}
	for _, route := range []struct {
		pattern, role string
		handler       http.HandlerFunc
	}{
		{"GET /api/tasks", "intern", a.tasks}, {"POST /api/tasks", "intern", a.create}, {"POST /api/tasks/{taskID}/complete", "intern", a.complete},
		{"GET /api/progress", "intern", a.progress}, {"GET /api/interns", "mentor", a.interns}, {"GET /api/interns/{id}/progress", "mentor", a.mentorProgress},
	} {
		browser.Handle(route.pattern, a.Auth.RequireRoles(route.handler, route.role))
		method, path, _ := strings.Cut(route.pattern, " ")
		browserMethods[path] = append(browserMethods[path], method)
	}
	jsonFallbacks(browser, browserMethods)
	mux.Handle("/api/", a.Auth.WithSession(browser))
	for _, route := range []struct {
		pattern string
		ticket  bool
		handler http.HandlerFunc
	}{
		{"GET /internal/interns/{id}/progress", false, a.progress}, {"POST /internal/interns/{id}/tasks", false, a.create},
		{"POST /internal/interns/{id}/tasks/{taskID}/complete", false, a.complete}, {"GET /internal/interns/{id}/mentor", true, a.mentor},
	} {
		mux.Handle(route.pattern, a.keyAuth(route.handler, route.ticket))
		method, path, _ := strings.Cut(route.pattern, " ")
		internalMethods[path] = append(internalMethods[path], method)
	}
	jsonFallbacks(mux, internalMethods)
	return mux
}

// Override ServeMux's text errors for unmatched paths and unsupported methods.
func jsonFallbacks(mux *http.ServeMux, methods map[string][]string) {
	for path, allowed := range methods {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Allow", strings.Join(allowed, ", "))
			jsonResponse(w, 405, map[string]string{"error": "method not allowed"})
		})
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 404, map[string]string{"error": "not found"})
	})
}
func (a *API) keyAuth(next http.Handler, ticketAllowed bool) http.Handler {
	buddy, ticket := sha256.Sum256([]byte(a.BuddyKey)), sha256.Sum256([]byte(a.TicketKey))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			jsonResponse(w, 401, map[string]string{"error": "unauthorized"})
			return
		}
		token := sha256.Sum256([]byte(parts[1]))
		isBuddy := subtle.ConstantTimeCompare(token[:], buddy[:]) == 1 && a.BuddyKey != ""
		isTicket := subtle.ConstantTimeCompare(token[:], ticket[:]) == 1 && a.TicketKey != ""
		if !isBuddy && !isTicket {
			jsonResponse(w, 401, map[string]string{"error": "unauthorized"})
			return
		}
		if !isBuddy && !ticketAllowed {
			jsonResponse(w, 403, map[string]string{"error": "forbidden"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
func owner(r *http.Request) string {
	if strings.HasPrefix(r.URL.Path, "/internal/") {
		return r.PathValue("id")
	}
	u, _ := auth.CurrentUser(r.Context())
	return u.ID
}
func (a *API) tasks(w http.ResponseWriter, r *http.Request) {
	v, e := a.Service.Tasks(r.Context(), owner(r))
	if e != nil {
		failure(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"tasks": v})
}
func (a *API) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	var in Input
	if d.Decode(&in) != nil || d.Decode(new(any)) != io.EOF {
		failure(w, ErrInvalid)
		return
	}
	v, created, e := a.Service.Create(r.Context(), owner(r), r.Header.Get("Idempotency-Key"), in)
	if e != nil {
		failure(w, e)
		return
	}
	code := 200
	if created {
		code = 201
	}
	jsonResponse(w, code, v)
}
func (a *API) complete(w http.ResponseWriter, r *http.Request) {
	v, e := a.Service.Complete(r.Context(), owner(r), r.PathValue("taskID"))
	if e != nil {
		failure(w, e)
		return
	}
	jsonResponse(w, 200, v)
}
func (a *API) progress(w http.ResponseWriter, r *http.Request) {
	v, e := a.Service.Progress(r.Context(), owner(r))
	if e != nil {
		failure(w, e)
		return
	}
	jsonResponse(w, 200, v)
}
func (a *API) mentorProgress(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.CurrentUser(r.Context())
	v, e := a.Service.MentorProgress(r.Context(), u.ID, r.PathValue("id"))
	if e != nil {
		failure(w, e)
		return
	}
	jsonResponse(w, 200, v)
}
func (a *API) interns(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.CurrentUser(r.Context())
	v, e := a.Service.Store.Interns(r.Context(), u.ID)
	if e != nil {
		failure(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"interns": v})
}
func (a *API) mentor(w http.ResponseWriter, r *http.Request) {
	v, e := a.Service.Mentor(r.Context(), owner(r))
	if e != nil {
		failure(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"mentor": v})
}
