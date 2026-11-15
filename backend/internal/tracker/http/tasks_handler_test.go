package trackerhttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-buddy/internal/tracker/apperror"
	"ai-buddy/internal/tracker/tasks"
	"ai-buddy/internal/users"
)

type tasksStub struct {
	err             error
	created         bool
	id, key, taskID string
	input           tasks.CreateInput
	calls           int
}

func (s *tasksStub) Create(_ context.Context, id, key string, input tasks.CreateInput) (tasks.Task, bool, error) {
	s.id = id
	s.key = key
	s.input = input
	s.calls++
	return tasks.Task{
		ID:          "task",
		InternID:    id,
		Title:       input.Title,
		Description: input.Description,
		Deadline:    input.Deadline,
	}, s.created, s.err
}

func (s *tasksStub) Complete(_ context.Context, id, taskID string) (tasks.Task, error) {
	s.id = id
	s.taskID = taskID
	s.calls++
	return tasks.Task{ID: taskID, InternID: id}, s.err
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
	h := New(service, nil, nil)
	input := `{"title":"Task","description":"Details","deadline":"2026-12-01"}`
	w := request(h.Create, input)
	var task tasks.Task
	err := json.Unmarshal(w.Body.Bytes(), &task)
	if err != nil ||
		w.Code != 201 ||
		task.ID != "task" ||
		task.InternID != "i" ||
		service.key != "key" ||
		service.input != (tasks.CreateInput{
			Title:       "Task",
			Description: "Details",
			Deadline:    "2026-12-01",
		}) {
		t.Fatal(w, service, err)
	}

	service.created = false
	w = request(h.Create, input)
	if w.Code != 200 {
		t.Fatal(w)
	}

	w = request(h.Complete, "")
	if w.Code != 200 || service.id != "i" || service.taskID != "task" {
		t.Fatal(w, service)
	}

	for _, tc := range []struct {
		err  error
		code int
	}{
		{apperror.ErrInvalid, 400},
		{apperror.ErrNotFound, 404},
		{apperror.ErrConflict, 409},
		{errors.New("SQL secret"), 500},
	} {
		service.err = tc.err
		for _, handler := range []http.HandlerFunc{h.Create, h.Complete} {
			w = request(handler, input)
			if w.Code != tc.code ||
				!strings.Contains(w.Header().Get("Content-Type"), "application/json") ||
				strings.Contains(w.Body.String(), "secret") {
				t.Fatal(w)
			}
		}
	}
}
