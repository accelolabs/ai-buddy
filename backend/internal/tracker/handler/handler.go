package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	// Project packages.
	"backend/internal/tracker"
	"backend/internal/tracker/service"
	"backend/internal/users"
	"backend/internal/users/middleware"
)

type taskService interface {
	Create(context.Context, users.User, string, string) (tracker.Task, error)
	List(context.Context, users.User) ([]tracker.Task, error)
	ListIntern(context.Context, users.User, string) ([]tracker.Task, error)
	Complete(context.Context, users.User, string) (tracker.Task, error)
}

type Handler struct {
	service taskService
}

type taskRequest struct {
	Description string `json:"description"`
	Deadline    string `json:"deadline"`
}

type taskResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Description string    `json:"description"`
	Deadline    time.Time `json:"deadline"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewHandler(service taskService) *Handler {
	return &Handler{
		service: service,
	}
}

func taskView(value tracker.Task) taskResponse {
	return taskResponse{
		ID:          value.ID,
		UserID:      value.UserID,
		Description: value.Description,
		Deadline:    value.Deadline,
		Status:      value.Status,
		CreatedAt:   value.CreatedAt,
	}
}

func selectedUser(w http.ResponseWriter, r *http.Request, role string) (users.User, bool) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "select a user", http.StatusUnauthorized)

		return users.User{}, false
	}

	if user.Role != role {
		http.Error(w, "role is not allowed", http.StatusForbidden)

		return users.User{}, false
	}

	return user, true
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, service.ErrForbidden) {
		http.Error(w, "role is not allowed", http.StatusForbidden)

		return
	}

	if errors.Is(err, service.ErrNotFound) {
		http.NotFound(w, r)

		return
	}

	log.Printf("tracker: %v", err)
	http.Error(w, "failed to access tasks", http.StatusInternalServerError)
}

func writeResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write tasks: %v", err)
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := selectedUser(w, r, users.RoleIntern)
	if !ok {
		return
	}

	values, err := h.service.List(r.Context(), user)
	h.list(w, r, values, err)
}

func (h *Handler) ListIntern(w http.ResponseWriter, r *http.Request) {
	user, ok := selectedUser(w, r, users.RoleMentor)
	if !ok {
		return
	}

	values, err := h.service.ListIntern(r.Context(), user, r.PathValue("internID"))
	h.list(w, r, values, err)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, values []tracker.Task, err error) {
	if err != nil {
		writeError(w, r, err)

		return
	}

	response := make([]taskResponse, 0, len(values))

	for _, value := range values {
		response = append(response, taskView(value))
	}

	writeResponse(w, http.StatusOK, response)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := selectedUser(w, r, users.RoleIntern)
	if !ok {
		return
	}

	var input taskRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid task", http.StatusBadRequest)

		return
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		http.Error(w, "invalid task", http.StatusBadRequest)

		return
	}

	value, err := h.service.Create(r.Context(), user, input.Description, input.Deadline)
	if errors.Is(err, service.ErrInvalidInput) {
		http.Error(w, "invalid task", http.StatusBadRequest)

		return
	}

	if errors.Is(err, service.ErrForbidden) {
		http.Error(w, "role is not allowed", http.StatusForbidden)

		return
	}

	if err != nil {
		log.Printf("create task: %v", err)
		http.Error(w, "failed to create task", http.StatusInternalServerError)

		return
	}

	writeResponse(w, http.StatusCreated, taskView(value))
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	user, ok := selectedUser(w, r, users.RoleIntern)
	if !ok {
		return
	}
	// Completion is a button action with no request body.
	var body [1]byte
	if n, err := r.Body.Read(body[:]); n != 0 || err != io.EOF {
		http.Error(w, "completion requires an empty body", http.StatusBadRequest)

		return
	}

	value, err := h.service.Complete(r.Context(), user, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)

		return
	}

	writeResponse(w, http.StatusOK, taskView(value))
}
