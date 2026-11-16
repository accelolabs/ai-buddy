package mcp

import (
	"context"
	"fmt"
	"strings"

	// Project packages.
	"backend/internal/buddy"
	"backend/internal/users"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type TicketArgs struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type TicketResult struct {
	Prepared bool `json:"prepared"`
}

func (h *Handler) PrepareTicket(
	ctx context.Context,
	_ *sdk.CallToolRequest,
	args TicketArgs,
) (*sdk.CallToolResult, TicketResult, error) {
	if h.user.Role != users.RoleIntern {
		return nil, TicketResult{}, fmt.Errorf("only interns can prepare tickets")
	}

	args.Title = strings.TrimSpace(args.Title)
	args.Description = strings.TrimSpace(args.Description)
	if args.Title == "" || args.Description == "" {
		return nil, TicketResult{}, fmt.Errorf("title and description are required")
	}

	mentor, err := h.users.GetMentor(ctx, h.user.ID)
	if err != nil {
		return nil, TicketResult{}, fmt.Errorf("get assigned mentor: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.ticket = &buddy.Ticket{
		Title:       args.Title,
		Description: args.Description,
		Recipient: buddy.Recipient{
			ID:   mentor.ID,
			Name: mentor.Name,
		},
	}

	return nil, TicketResult{Prepared: true}, nil
}

func (h *Handler) Ticket() *buddy.Ticket {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.ticket == nil {
		return nil
	}

	value := *h.ticket

	return &value
}
