package ticket

import (
	"errors"
	"net/http"

	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/users"
)

func (h *Service) CreateHTTP(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := httpx.Decode(w, r, &input); err != nil {
		Failure(w, err)
		return
	}
	ticket, created, err := h.Create(
		r.Context(),
		users.CurrentUser(r.Context()).ID,
		r.Header.Get("Idempotency-Key"),
		input,
	)
	if err != nil {
		Failure(w, err)
		return
	}
	code := http.StatusOK
	if created {
		code = http.StatusCreated
	}
	httpx.JSON(w, code, ticket)
}

type ticketsResponse struct {
	Tickets []Ticket `json:"tickets"`
}

func (h *Service) IncomingHTTP(w http.ResponseWriter, r *http.Request) {
	list, err := h.Incoming(r.Context(), users.CurrentUser(r.Context()).ID)
	if err != nil {
		Failure(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ticketsResponse{Tickets: list})
}

func Register(mux *http.ServeMux, login *users.Selector, h *Service) {
	mux.Handle("GET /api/tickets", login.RequireRoles(http.HandlerFunc(h.SentHTTP), users.RoleIntern))
	mux.Handle("POST /api/tickets", login.RequireRoles(http.HandlerFunc(h.CreateHTTP), users.RoleIntern))
	mux.Handle("GET /api/tickets/incoming", login.RequireRoles(http.HandlerFunc(h.IncomingHTTP), users.RoleMentor))
}

func (h *Service) SentHTTP(w http.ResponseWriter, r *http.Request) {
	list, err := h.Sent(r.Context(), users.CurrentUser(r.Context()).ID)
	if err != nil {
		Failure(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ticketsResponse{Tickets: list})
}

func Failure(w http.ResponseWriter, err error) {
	code, message := 500, "storage unavailable"
	switch {
	case errors.Is(err, ErrNoMentor):
		code, message = 422, ErrNoMentor.Error()
	case errors.Is(err, ErrInvalid), errors.Is(err, httpx.ErrInvalid):
		code, message = 400, "invalid input"
	case errors.Is(err, ErrNotFound):
		code, message = 404, "not found"
	case errors.Is(err, ErrConflict):
		code, message = 409, ErrConflict.Error()
	}

	httpx.JSON(w, code, map[string]string{"error": message})
}
