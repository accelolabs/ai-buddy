package buddy

import (
	"context"
	"encoding/json"
	"strings"

	"ai-buddy/internal/ticket"
	"ai-buddy/internal/tracker"
	"ai-buddy/internal/users"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type runTools struct {
	tools   *Tools
	user    users.User
	run     Run
	request string
}

func (h *runTools) search(ctx context.Context, _ *sdk.CallToolRequest, a searchArgs) (*sdk.CallToolResult, any, error) {

	if strings.TrimSpace(a.Query) == "" || len([]rune(a.Query)) > 4000 {
		return nil, nil, ErrInvalid
	}
	sources, err := h.tools.Wiki.Search(ctx, h.user.ID, a.Query)
	if err != nil {
		return nil, nil, err
	}
	for _, source := range sources {
		if err = h.tools.Repo.Event(ctx, h.run.ID, EventSource, source); err != nil {
			return nil, nil, err
		}
	}
	if len(sources) == 0 {
		p, e := h.tools.Tracker.Progress(ctx, h.user.ID)
		if e != nil {
			return nil, nil, e
		}
		return nil, map[string]any{"sources": sources, "message": "No answer found. Offer to contact the mentor.", "mentor_assigned": p.Mentor != nil}, nil
	}
	return nil, sources, nil
}

func (h *runTools) progress(ctx context.Context, _ *sdk.CallToolRequest, _ emptyArgs) (*sdk.CallToolResult, any, error) {

	p, e := h.tools.Tracker.Progress(ctx, h.user.ID)
	return nil, p, e
}

func (h *runTools) tickets(ctx context.Context, _ *sdk.CallToolRequest, _ emptyArgs) (*sdk.CallToolResult, any, error) {

	p, e := h.tools.Ticket.Sent(ctx, h.user.ID)
	return nil, p, e
}

func (h *runTools) draft(ctx context.Context, kind string, fields any) (*sdk.CallToolResult, any, error) {

	b, e := json.Marshal(fields)
	if e != nil {
		return nil, nil, e
	}
	if _, e = parseFields(kind, b); e != nil {
		return nil, nil, e
	}
	a, e := h.tools.Repo.Draft(ctx, Action{RunID: h.run.ID, Owner: h.user.ID, Kind: kind, Fields: b})
	if e != nil {
		return nil, nil, e
	}
	card := map[string]any{"id": a.ID, "kind": a.Kind, "status": a.Status, "fields": a.Fields}
	if e = h.tools.Repo.Event(ctx, h.run.ID, EventAction, card); e != nil {
		return nil, nil, e
	}
	return nil, card, nil
}

func (h *runTools) createTask(ctx context.Context, _ *sdk.CallToolRequest, a tracker.CreateInput) (*sdk.CallToolResult, any, error) {
	return h.draft(ctx, "create_task", a)
}

func (h *runTools) createTicket(ctx context.Context, _ *sdk.CallToolRequest, a ticket.CreateInput) (*sdk.CallToolResult, any, error) {
	return h.draft(ctx, "create_ticket", a)
}

func (h *runTools) complete(ctx context.Context, _ *sdk.CallToolRequest, a completeArgs) (*sdk.CallToolResult, any, error) {

	p, e := h.tools.Tracker.Progress(ctx, h.user.ID)
	if e != nil {
		return nil, nil, e
	}
	if !CanComplete(h.request, a.TaskID, p.Tasks) {
		return nil, map[string]string{"clarification": "Ask the user to explicitly identify one task to complete"}, nil
	}
	r, e := h.tools.Tracker.CompleteTask(ctx, h.user.ID, a.TaskID)
	if e == nil {
		e = h.tools.Repo.Event(ctx, h.run.ID, EventResult, r)
	}
	return nil, r, e
}
