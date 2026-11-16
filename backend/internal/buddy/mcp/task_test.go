package mcp

import (
	"encoding/json"
	"testing"
	"time"

	// Project packages.
	"backend/internal/users"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPrepareTask(t *testing.T) {
	h := NewHandler(nil, users.User{Role: users.RoleIntern}, nil)
	session, closeSession, err := h.OpenSession(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	defer closeSession()

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	var names []string

	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}

	encoded, err := json.Marshal(names)
	if err != nil || string(encoded) != `["attach_source","prepare_task","prepare_ticket","search_documents"]` {
		t.Fatalf("tools: %s %v", encoded, err)
	}

	call := func(args map[string]any, fail bool) {
		t.Helper()
		result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
			Name:      "prepare_task",
			Arguments: args,
		})
		if err != nil || result.IsError != fail {
			t.Fatalf("call: %+v %v", result, err)
		}

		if !fail {
			data, err := json.Marshal(result.StructuredContent)
			if err != nil || string(data) != `{"prepared":true}` {
				t.Fatalf("result: %s %v", data, err)
			}
		}
	}
	call(map[string]any{
		"description": " \nЗадача 🔑\t",
		"deadline":    "2020-01-01T12:00:00+03:00",
	}, false)
	want := *h.Task()
	if want.Description != "Задача 🔑" || want.Deadline.Format(time.RFC3339) != "2020-01-01T12:00:00+03:00" {
		t.Fatal(want)
	}

	for _, args := range []map[string]any{
		{},
		{"description": "x"},
		{"deadline": "2026-01-01T00:00:00Z"},
		{
			"description": 1,
			"deadline":    "2026-01-01T00:00:00Z",
		},
		{
			"description": "\u2003",
			"deadline":    "2026-01-01T00:00:00Z",
		},
		{
			"description": "x",
			"deadline":    "",
		},
		{
			"description": "x",
			"deadline":    "2026-01-01",
		},
		{
			"description": "x",
			"deadline":    "2026-01-01T12:00:00",
		},
		{
			"description": "x",
			"deadline":    "2026-02-30T12:00:00Z",
		},
	} {
		call(args, true)
		if *h.Task() != want {
			t.Fatal("error replaced draft")
		}
	}

	call(map[string]any{
		"description": "Новая",
		"deadline":    "2030-01-01T00:00:00Z",
	}, false)
	copy := h.Task()
	if copy.Description != "Новая" {
		t.Fatal(copy)
	}

	copy.Description = "mutated"
	if h.Task().Description != "Новая" {
		t.Fatal("mutable draft")
	}

	if NewHandler(nil, users.User{Role: users.RoleIntern}, nil).Task() != nil {
		t.Fatal("draft leaked")
	}

	for _, role := range []string{users.RoleMentor, ""} {
		other := NewHandler(nil, users.User{Role: role}, nil)
		if _, _, err := other.PrepareTask(t.Context(), nil, TaskArgs{
			Description: "x",
			Deadline:    "2030-01-01T00:00:00Z",
		}); err == nil ||
			other.Task() != nil {
			t.Fatal("role bypass")
		}

		session, closeSession, err := other.OpenSession(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		listed, err := session.ListTools(t.Context(), nil)
		closeSession()
		if err != nil {
			t.Fatal(err)
		}

		for _, tool := range listed.Tools {
			if tool.Name == "prepare_task" {
				t.Fatal("task tool exposed")
			}
		}
	}
}
