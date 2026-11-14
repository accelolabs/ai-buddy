package taskshttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-buddy/backend-tracker/internal/apperror"
	"ai-buddy/backend-tracker/internal/tasks"
)

type serviceStub struct {
	err             error
	created         bool
	id, key, taskID string
	input           tasks.CreateInput
	calls           int
}

func (s *serviceStub) Tasks(_ context.Context, id string) ([]tasks.Task, error) {
	s.id = id
	s.calls++
	return []tasks.Task{{ID: "task", InternID: id}}, s.err
}

func (s *serviceStub) Create(_ context.Context, id, key string, input tasks.CreateInput) (tasks.Task, bool, error) {
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

func (s *serviceStub) Complete(_ context.Context, id, taskID string) (tasks.Task, error) {
	s.id = id
	s.taskID = taskID
	s.calls++
	return tasks.Task{ID: taskID, InternID: id}, s.err
}

func request(handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/internal/interns/i/tasks", strings.NewReader(body))
	r.SetPathValue("id", "i")
	r.SetPathValue("taskID", "task")
	r.Header.Set("Idempotency-Key", "key")
	w := httptest.NewRecorder()
	handler(w, r)
	return w
}

func TestHandlers(t *testing.T) {
	service := &serviceStub{created: true}
	h := New(service)
	w := request(h.List, "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"tasks":[`) || service.id != "i" {
		t.Fatal(w, service)
	}

	input := `{"title":"Task","description":"Details","deadline":"2026-12-01"}`
	w = request(h.Create, input)
	var task taskResponse
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
		for _, handler := range []http.HandlerFunc{h.List, h.Create, h.Complete} {
			w = request(handler, input)
			if w.Code != tc.code ||
				!strings.Contains(w.Header().Get("Content-Type"), "application/json") ||
				strings.Contains(w.Body.String(), "secret") {
				t.Fatal(w)
			}
		}
	}
}

func TestCreateDecodeErrors(t *testing.T) {
	service := &serviceStub{}
	h := New(service)
	for _, body := range []string{
		"",
		`{`,
		`[]`,
		`{"extra":1}`,
		`{} {}`,
		strings.Repeat("x", (1<<20)+1),
	} {
		w := request(h.Create, body)
		if w.Code != 400 || service.calls != 0 {
			t.Fatal(w, service)
		}
	}
}
