package mcp

import (
	"errors"
	"testing"

	// Project packages.
	"backend/internal/tracker"
	"backend/internal/users"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPrepareTaskCompletion(t *testing.T) {
	user := users.User{ID: "trusted", Role: users.RoleIntern}
	store := &taskStore{values: []tracker.Task{
		{ID: "first", UserID: user.ID, Description: "Первая", Status: "pending"},
		{ID: "second", UserID: user.ID, Description: "Вторая", Status: "completed"},
	}}
	h := NewHandler(nil, user, nil, store)
	if h.TaskCompletion() != nil {
		t.Fatal("card exists before tool call")
	}

	session, closeSession, err := h.OpenSession(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	defer closeSession()

	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: "prepare_task_completion", Arguments: map[string]any{"task_id": "first"},
	})
	if err != nil || result.IsError || h.TaskCompletion() == nil || h.TaskCompletion().ID != "first" ||
		store.user != user || store.values[0].Status != "pending" {
		t.Fatalf("completion: %+v; %v", result, err)
	}

	card := h.TaskCompletion()
	card.Description = "changed"
	if h.TaskCompletion().Description != "Первая" {
		t.Fatal("card mutated by caller")
	}

	store.values[0].Description = "changed in service"
	if h.TaskCompletion().Description != "Первая" {
		t.Fatal("card mutated by service")
	}

	for _, id := range []string{"", "foreign", "missing"} {
		if _, _, err := h.PrepareTaskCompletion(t.Context(), nil, TaskCompletionArgs{TaskID: id}); err == nil {
			t.Fatalf("accepted unavailable task %q", id)
		}

		if h.TaskCompletion().ID != "first" {
			t.Fatal("failed selection replaced card")
		}
	}

	store.err = errors.New("database error")
	_, _, err = h.PrepareTaskCompletion(t.Context(), nil, TaskCompletionArgs{TaskID: "first"})
	if !errors.Is(err, store.err) {
		t.Fatal(err)
	}

	if h.TaskCompletion().ID != "first" {
		t.Fatal("service failure lost card")
	}

	store.err = nil
	_, _, err = h.PrepareTaskCompletion(t.Context(), nil, TaskCompletionArgs{TaskID: "second"})
	if err != nil || h.TaskCompletion().ID != "second" {
		t.Fatalf("replacement: %v", err)
	}

	if NewHandler(nil, user, nil, store).TaskCompletion() != nil {
		t.Fatal("card leaked to another run")
	}
}

func TestTaskCompletionRole(t *testing.T) {
	for _, role := range []string{users.RoleMentor, ""} {
		store := &taskStore{}
		h := NewHandler(nil, users.User{Role: role}, nil, store)
		_, _, err := h.PrepareTaskCompletion(t.Context(), nil, TaskCompletionArgs{TaskID: "task"})
		if err == nil || store.calls != 0 {
			t.Fatal("forbidden call reached service")
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
			if tool.Name == "prepare_task_completion" {
				t.Fatalf("tool exposed to %q", role)
			}
		}
	}
}
