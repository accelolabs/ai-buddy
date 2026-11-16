package mcp

import (
	"context"
	"fmt"

	// Project packages.
	"backend/internal/tracker"
	"backend/internal/users"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type TaskCompletionArgs struct {
	TaskID string `json:"task_id"`
}

type TaskCompletionResult struct {
	Prepared bool `json:"prepared"`
}

func (h *Handler) PrepareTaskCompletion(
	ctx context.Context,
	_ *sdk.CallToolRequest,
	args TaskCompletionArgs,
) (*sdk.CallToolResult, TaskCompletionResult, error) {
	if h.user.Role != users.RoleIntern {
		return nil, TaskCompletionResult{}, fmt.Errorf("only interns can prepare task completion")
	}

	if args.TaskID == "" {
		return nil, TaskCompletionResult{}, fmt.Errorf("task_id is required")
	}

	values, err := h.tracker.List(ctx, h.user)
	if err != nil {
		return nil, TaskCompletionResult{}, fmt.Errorf("list tasks for completion: %w", err)
	}

	for _, value := range values {
		if value.ID == args.TaskID {
			h.mu.Lock()
			defer h.mu.Unlock()

			h.taskCompletion = &value

			return nil, TaskCompletionResult{Prepared: true}, nil
		}
	}

	return nil, TaskCompletionResult{}, fmt.Errorf("task is not available")
}

func (h *Handler) TaskCompletion() *tracker.Task {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.taskCompletion == nil {
		return nil
	}

	value := *h.taskCompletion

	return &value
}
