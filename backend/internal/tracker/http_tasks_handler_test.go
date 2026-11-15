package tracker

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-buddy/internal/users"
)

type tasksStub struct {
	err             error
	created         bool
	id, key, taskID string
	input           CreateInput
	calls           int
}

func (s *tasksStub) Create(_ context.Context, id, key string, input CreateInput) (Task, bool, error) {
	s.id = id
	s.key = key
	s.input = input
	s.calls++
	return Task{
		ID:          "task",
		InternID:    id,
		Title:       input.Title,
		Description: input.Description,
		Deadline:    input.Deadline,
	}, s.created, s.err
}

func (s *tasksStub) Complete(_ context.Context, id, taskID string) (Task, error) {
	s.id = id
	s.taskID = taskID
	s.calls++
	return Task{ID: taskID, InternID: id}, s.err
}

func request(handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/api/tasks", strings.NewReader(body))
	r = r.WithContext(users.WithUser(r.Context(), users.User{ID: "i", Role: "intern", Team: "backend"}))
	r.SetPathValue("taskID", "task")
	r.Header.Set("Idempotency-Key", "key")
	w := httptest.NewRecorder()
	handler(w, r)
	return w
}

func TestTasksHandlers(t *testing.T) {
	service := &tasksStub{created: true}
	h := &Service{Store: service, Location: time.UTC, Now: time.Now}
	input := `{"title":"Task","description":"Details","deadline":"2026-12-01"}`
	w := request(h.CreateHTTP, input)
	var task Task
	err := json.Unmarshal(w.Body.Bytes(), &task)
	if err != nil ||
		w.Code != 201 ||
		task.ID != "task" ||
		task.InternID != "i" ||
		service.key != "key" ||
		service.input != (CreateInput{
			Title:       "Task",
			Description: "Details",
			Deadline:    "2026-12-01",
		}) {
		t.Fatal(w, service, err)
	}

	service.created = false
	w = request(h.CreateHTTP, input)
	if w.Code != 200 {
		t.Fatal(w)
	}

	w = request(h.CompleteHTTP, "")
	if w.Code != 200 || service.id != "i" || service.taskID != "task" {
		t.Fatal(w, service)
	}

	for _, tc := range []struct {
		err  error
		code int
	}{
		{ErrInvalid, 400},
		{ErrNotFound, 404},
		{ErrConflict, 409},
		{errors.New("SQL secret"), 500},
	} {
		service.err = tc.err
		for _, handler := range []http.HandlerFunc{h.CreateHTTP, h.CompleteHTTP} {
			w = request(handler, input)
			if w.Code != tc.code ||
				!strings.Contains(w.Header().Get("Content-Type"), "application/json") ||
				strings.Contains(w.Body.String(), "secret") {
				t.Fatal(w)
			}
		}
	}
}

func (s *tasksStub) Tasks(context.Context, string) ([]Task, error) { panic("unused") }
