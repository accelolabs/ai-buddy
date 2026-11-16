package mcp

import (
	"backend/internal/buddy"
	"backend/internal/users"
	"context"
	"fmt"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type TaskArgs struct {
	Description string `json:"description"`
	Deadline    string `json:"deadline"`
}

type TaskResult struct {
	Prepared bool `json:"prepared"`
}

func (h *Handler) PrepareTask(_ context.Context, _ *sdk.CallToolRequest, args TaskArgs) (*sdk.CallToolResult, TaskResult, error) {
	if h.user.Role != users.RoleIntern {
		return nil, TaskResult{}, fmt.Errorf("only interns can prepare tasks")
	}
	description := strings.TrimSpace(args.Description)
	deadline, err := time.Parse(time.RFC3339, args.Deadline)
	if description == "" || err != nil {
		return nil, TaskResult{}, fmt.Errorf("description and RFC3339 deadline with timezone are required")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.task = &buddy.Task{Description: description, Deadline: deadline}
	return nil, TaskResult{Prepared: true}, nil
}

func (h *Handler) Task() *buddy.Task {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.task == nil {
		return nil
	}
	value := *h.task
	return &value
}
