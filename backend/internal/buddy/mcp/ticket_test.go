package mcp

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	// Project packages.
	"backend/internal/buddy"
	"backend/internal/users"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type mentorStore struct {
	user   users.User
	err    error
	intern string
}

func (m *mentorStore) GetMentor(_ context.Context, id string) (users.User, error) {
	m.intern = id

	return m.user, m.err
}

func TestPrepareTicket(t *testing.T) {
	store := &mentorStore{user: users.User{
		ID:   "mentor",
		Name: "Ментор",
	}}
	user := users.User{
		ID:   "intern",
		Role: users.RoleIntern,
	}

	h := NewHandler(&fakeRAG{}, user, store, nil)
	session, closeSession, err := h.OpenSession(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	defer closeSession()

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil || len(tools.Tools) != 6 {
		t.Fatalf("tools: %v %v", tools, err)
	}

	call := func(args map[string]any, fail bool) {
		t.Helper()
		result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
			Name:      "prepare_ticket",
			Arguments: args,
		})
		if err != nil || result.IsError != fail {
			t.Fatalf("result: %+v %v", result, err)
		}
	}

	call(map[string]any{
		"title":       "  Тема 🔑 ",
		"description": "\nОписание\t",
	}, false)
	want := &buddy.Ticket{
		Title:       "Тема 🔑",
		Description: "Описание",
		Recipient: buddy.Recipient{
			ID:   "mentor",
			Name: "Ментор",
		},
	}

	if !reflect.DeepEqual(h.Ticket(), want) || store.intern != user.ID {
		t.Fatalf("card: %+v", h.Ticket())
	}

	for _, args := range []map[string]any{
		{}, {"title": "x"}, {
			"title":       1,
			"description": "y",
		}, {
			"title":       " \t",
			"description": "y",
		}, {
			"title":       "x",
			"description": "\u2003",
		},
	} {
		call(args, true)
		if !reflect.DeepEqual(h.Ticket(), want) {
			t.Fatal("failed call replaced draft")
		}
	}

	store.err = sql.ErrNoRows
	call(map[string]any{
		"title":       "new",
		"description": "new",
	}, true)
	if !reflect.DeepEqual(h.Ticket(), want) {
		t.Fatal("missing mentor replaced draft")
	}

	store.err = nil
	call(map[string]any{
		"title":       "new",
		"description": "new",
	}, false)
	if h.Ticket().Title != "new" {
		t.Fatal("draft was not replaced")
	}

	copy := h.Ticket()
	copy.Title = "changed"
	if h.Ticket().Title != "new" {
		t.Fatal("caller modified draft")
	}

	other := NewHandler(&fakeRAG{}, user, store, nil)
	if other.Ticket() != nil {
		t.Fatal("draft leaked across runs")
	}

	store.err = sql.ErrNoRows
	if _, _, err := other.PrepareTicket(t.Context(), nil, TicketArgs{
		Title:       "x",
		Description: "y",
	}); err == nil ||
		other.Ticket() != nil {
		t.Fatal("missing mentor accepted")
	}

	mentor := NewHandler(&fakeRAG{}, users.User{Role: users.RoleMentor}, store, nil)
	if _, _, err := mentor.PrepareTicket(t.Context(), nil, TicketArgs{
		Title:       "x",
		Description: "y",
	}); err == nil {
		t.Fatal("mentor prepared ticket")
	}

	ms, closeMentor, err := mentor.OpenSession(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	defer closeMentor()

	listed, err := ms.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, tool := range listed.Tools {
		if tool.Name == "prepare_ticket" {
			t.Fatal("tool exposed to mentor")
		}
	}
}
