package mcp

import (
	"context"
	"fmt"
	"slices"
	"time"

	// Project packages.
	"backend/internal/tracker"
	"backend/internal/users"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListTasksArgs struct{}

type ListTasksResult struct {
	Tasks []ListedTask `json:"tasks"`
}

type ListedTask struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Description string    `json:"description"`
	Deadline    time.Time `json:"deadline"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *Handler) ListTasks(
	ctx context.Context,
	_ *sdk.CallToolRequest,
	_ ListTasksArgs,
) (*sdk.CallToolResult, ListTasksResult, error) {
	if h.user.Role != users.RoleIntern {
		return nil, ListTasksResult{}, fmt.Errorf("only interns can list their tasks")
	}

	values, err := h.tracker.List(ctx, h.user)
	if err != nil {
		return nil, ListTasksResult{}, fmt.Errorf("list tasks: %w", err)
	}

	tasks := make([]tracker.Task, len(values))
	copy(tasks, values)
	result := ListTasksResult{Tasks: make([]ListedTask, 0, len(tasks))}

	for _, value := range tasks {
		result.Tasks = append(result.Tasks, ListedTask{
			ID:          value.ID,
			UserID:      value.UserID,
			Description: value.Description,
			Deadline:    value.Deadline,
			Status:      value.Status,
			CreatedAt:   value.CreatedAt,
		})
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.tasks = tasks

	return nil, result, nil
}

func (h *Handler) Tasks() []tracker.Task {
	h.mu.Lock()
	defer h.mu.Unlock()

	return slices.Clone(h.tasks)
}
