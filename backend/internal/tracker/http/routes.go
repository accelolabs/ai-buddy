package trackerhttp

import (
	"ai-buddy/internal/users"
	"net/http"
)

func Register(mux *http.ServeMux, login *users.Selector, h *Handler) {
	for _, r := range []struct {
		pattern, role string
		handler       http.HandlerFunc
	}{
		{"POST /api/tasks", "intern", h.Create},
		{"POST /api/tasks/{taskID}/complete", "intern", h.Complete}, {"GET /api/progress", "intern", h.Progress},
		{"GET /api/interns", "mentor", h.ListInterns}, {"GET /api/interns/{id}/progress", "mentor", h.MentorProgress},
	} {
		mux.Handle(r.pattern, login.RequireRoles(r.handler, r.role))
	}
}
