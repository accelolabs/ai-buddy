package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	// Project packages.
	"backend/internal/tracker"
	"backend/internal/users"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type taskStore struct {
	values []tracker.Task
	err    error
	user   users.User
	calls  int
}

func (s *taskStore) List(_ context.Context, user users.User) ([]tracker.Task, error) {
	s.user = user
	s.calls++

	return s.values, s.err
}

func TestListTasks(t *testing.T) {
	user := users.User{ID: "trusted", Role: users.RoleIntern, Team: "backend"}
	store := &taskStore{values: []tracker.Task{{
		ID: "task", UserID: user.ID, Description: "Задача",
		Deadline: time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC), Status: "pending",
		CreatedAt: time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC),
	}}}
	h := NewHandler(nil, user, nil, store)
	if h.Tasks() != nil {
		t.Fatal("list attached before tool call")
	}

	session, closeSession, err := h.OpenSession(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	defer closeSession()

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: "list_tasks", Arguments: map[string]any{},
	})
	if err != nil || result.IsError {
		t.Fatalf("MCP result: %+v, %v", result, err)
	}

	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	var output ListTasksResult
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatal(err)
	}

	if store.user != user || len(output.Tasks) != 1 || output.Tasks[0].ID != "task" ||
		output.Tasks[0].UserID != user.ID || output.Tasks[0].Status != "pending" ||
		output.Tasks[0].Description != "Задача" || output.Tasks[0].Deadline != store.values[0].Deadline.Format(time.DateOnly) ||
		!output.Tasks[0].CreatedAt.Equal(store.values[0].CreatedAt) {
		t.Fatalf("tasks: %s; user: %+v", encoded, store.user)
	}

	expected := append([]tracker.Task(nil), store.values...)
	store.values[0].Description = "changed in service"
	snapshot := h.Tasks()
	snapshot[0].Description = "changed by caller"
	if !reflect.DeepEqual(h.Tasks(), expected) {
		t.Fatal("task snapshot was mutated")
	}

	store.err = errors.New("database failure")
	if _, _, err := h.ListTasks(t.Context(), nil, ListTasksArgs{}); !errors.Is(err, store.err) {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(h.Tasks(), expected) {
		t.Fatal("failed call lost previous card")
	}

	store.err = nil
	store.values = nil
	_, empty, err := h.ListTasks(t.Context(), nil, ListTasksArgs{})
	if err != nil || empty.Tasks == nil || len(empty.Tasks) != 0 || h.Tasks() == nil || len(h.Tasks()) != 0 {
		t.Fatalf("empty result: %+v; %v", empty, err)
	}

	if NewHandler(nil, user, nil, store).Tasks() != nil {
		t.Fatal("tasks leaked between runs")
	}
}

func TestListTasksRole(t *testing.T) {
	for _, role := range []string{users.RoleMentor, ""} {
		store := &taskStore{}
		h := NewHandler(nil, users.User{Role: role}, nil, store)
		if _, _, err := h.ListTasks(t.Context(), nil, ListTasksArgs{}); err == nil || store.calls != 0 {
			t.Fatal("forbidden list reached service")
		}

		session, closeSession, err := h.OpenSession(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		tools, err := session.ListTools(t.Context(), nil)
		closeSession()
		if err != nil {
			t.Fatal(err)
		}

		for _, tool := range tools.Tools {
			if tool.Name == "list_tasks" {
				t.Fatalf("tool exposed to %q", role)
			}
		}
	}
}
