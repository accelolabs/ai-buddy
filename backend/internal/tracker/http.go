package tracker

import (
	"errors"
	"net/http"

	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/users"
)

func Failure(w http.ResponseWriter, err error) {
	code, message := 500, "storage unavailable"
	switch {
	case errors.Is(err, ErrInvalid), errors.Is(err, httpx.ErrInvalid):
		code, message = 400, "invalid input"
	case errors.Is(err, ErrNotFound):
		code, message = 404, "not found"
	case errors.Is(err, ErrConflict):
		code, message = 409, ErrConflict.Error()
	}

	httpx.JSON(w, code, map[string]string{"error": message})
}

type internsResponse struct {
	Interns []users.User `json:"interns"`
}

func (h *Service) ListInterns(w http.ResponseWriter, r *http.Request) {
	u := users.CurrentUser(r.Context())
	result, err := h.Profiles.Interns(r.Context(), u.ID)
	if err != nil {
		Failure(w, err)
		return
	}

	httpx.JSON(w, 200, internsResponse{Interns: result})
}

func (h *Service) ProgressHTTP(w http.ResponseWriter, r *http.Request) {
	result, err := h.Progress(r.Context(), users.CurrentUser(r.Context()).ID)
	if err != nil {
		Failure(w, err)
		return
	}

	httpx.JSON(w, 200, result)
}

func (h *Service) MentorProgressHTTP(w http.ResponseWriter, r *http.Request) {
	u := users.CurrentUser(r.Context())
	result, err := h.MentorProgress(r.Context(), u.ID, r.PathValue("id"))
	if err != nil {
		Failure(w, err)
		return
	}

	httpx.JSON(w, 200, result)
}

func Register(mux *http.ServeMux, login *users.Selector, h *Service) {
	for _, r := range []struct {
		pattern, role string
		handler       http.HandlerFunc
	}{
		{"POST /api/tasks", "intern", h.CreateHTTP},
		{"POST /api/tasks/{taskID}/complete", "intern", h.CompleteHTTP}, {"GET /api/progress", "intern", h.ProgressHTTP},
		{"GET /api/interns", "mentor", h.ListInterns}, {"GET /api/interns/{id}/progress", "mentor", h.MentorProgressHTTP},
	} {
		mux.Handle(r.pattern, login.RequireRoles(r.handler, r.role))
	}
}

func (h *Service) CompleteHTTP(w http.ResponseWriter, r *http.Request) {
	result, err := h.CompleteTask(r.Context(), users.CurrentUser(r.Context()).ID, r.PathValue("taskID"))
	if err != nil {
		Failure(w, err)
		return
	}

	httpx.JSON(w, 200, result)
}

func (h *Service) CreateHTTP(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := httpx.Decode(w, r, &input); err != nil {
		Failure(w, err)
		return
	}

	result, created, err := h.CreateTask(r.Context(), users.CurrentUser(r.Context()).ID, r.Header.Get("Idempotency-Key"), input)
	if err != nil {
		Failure(w, err)
		return
	}

	code := 200
	if created {
		code = 201
	}

	httpx.JSON(w, code, result)
}
