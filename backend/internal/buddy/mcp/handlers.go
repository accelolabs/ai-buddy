// Package mcp hosts a private MCP server for each trusted run.
package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/platform/auth"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type runTools struct {
	tools   *Tools
	user    auth.User
	run     buddy.Run
	request string
}

func (h *runTools) search(ctx context.Context, _ *sdk.CallToolRequest, a searchArgs) (*sdk.CallToolResult, any, error) {
	t := h.tools
	u := h.user
	run := h.run

	if strings.TrimSpace(a.Query) == "" || len([]rune(a.Query)) > 4000 {
		return nil, nil, buddy.ErrInvalid
	}
	sources, err := t.Wiki.Search(ctx, u.ID, a.Query)
	if err != nil {
		return nil, nil, err
	}
	for _, source := range sources {
		if err = t.Repo.Event(ctx, run.ID, buddy.EventSource, source); err != nil {
			return nil, nil, err
		}
	}
	if len(sources) == 0 {
		p, e := t.Tracker.Progress(ctx, u.ID)
		if e != nil {
			return nil, nil, e
		}
		return nil, map[string]any{"sources": sources, "message": "No answer found. Offer to contact the mentor.", "mentor_assigned": len(p.Mentor) > 0 && string(p.Mentor) != "null"}, nil
	}
	return nil, sources, nil
}

func (h *runTools) progress(ctx context.Context, _ *sdk.CallToolRequest, _ emptyArgs) (*sdk.CallToolResult, any, error) {
	t := h.tools
	u := h.user

	p, e := t.Tracker.Progress(ctx, u.ID)
	return nil, p, e
}

func (h *runTools) tickets(ctx context.Context, _ *sdk.CallToolRequest, _ emptyArgs) (*sdk.CallToolResult, any, error) {
	t := h.tools
	u := h.user

	p, e := t.Ticket.List(ctx, u.ID)
	return nil, p, e
}

func (h *runTools) draft(ctx context.Context, kind string, fields any) (*sdk.CallToolResult, any, error) {
	t := h.tools
	u := h.user
	run := h.run

	b, e := json.Marshal(fields)
	if e != nil {
		return nil, nil, e
	}
	if e = buddy.ValidateFields(kind, b); e != nil {
		return nil, nil, e
	}
	a, e := t.Repo.Draft(ctx, buddy.Action{RunID: run.ID, Owner: u.ID, Kind: kind, Fields: b})
	if e != nil {
		return nil, nil, e
	}
	card := map[string]any{"id": a.ID, "kind": a.Kind, "status": a.Status, "fields": a.Fields}
	if e = t.Repo.Event(ctx, run.ID, buddy.EventAction, card); e != nil {
		return nil, nil, e
	}
	return nil, card, nil
}

func (h *runTools) createTask(ctx context.Context, _ *sdk.CallToolRequest, a buddy.TaskFields) (*sdk.CallToolResult, any, error) {
	return h.draft(ctx, "create_task", a)
}

func (h *runTools) createTicket(ctx context.Context, _ *sdk.CallToolRequest, a buddy.TicketFields) (*sdk.CallToolResult, any, error) {
	return h.draft(ctx, "create_ticket", a)
}

func (h *runTools) complete(ctx context.Context, _ *sdk.CallToolRequest, a completeArgs) (*sdk.CallToolResult, any, error) {
	t := h.tools
	u := h.user
	run := h.run
	request := h.request

	p, e := t.Tracker.Progress(ctx, u.ID)
	if e != nil {
		return nil, nil, e
	}
	if !CanComplete(request, a.TaskID, p.Tasks) {
		return nil, map[string]string{"clarification": "Ask the user to explicitly identify one task to complete"}, nil
	}
	r, e := t.Tracker.Complete(ctx, u.ID, a.TaskID)
	if e == nil {
		e = t.Repo.Event(ctx, run.ID, buddy.EventResult, r)
	}
	return nil, r, e
}
