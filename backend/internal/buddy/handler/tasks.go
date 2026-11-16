package handler

import (
	"time"

	// Project packages.
	"backend/internal/tracker"
)

type listedTaskResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Description string    `json:"description"`
	Deadline    time.Time `json:"deadline"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

func tasksView(tasks []tracker.Task) []listedTaskResponse {
	if tasks == nil {
		return nil
	}

	response := make([]listedTaskResponse, 0, len(tasks))

	for _, value := range tasks {
		response = append(response, listedTaskView(value))
	}

	return response
}

func listedTaskView(value tracker.Task) listedTaskResponse {
	return listedTaskResponse{
		ID:          value.ID,
		UserID:      value.UserID,
		Description: value.Description,
		Deadline:    value.Deadline,
		Status:      value.Status,
		CreatedAt:   value.CreatedAt,
	}
}

func taskCompletionView(value *tracker.Task) *listedTaskResponse {
	if value == nil {
		return nil
	}

	response := listedTaskView(*value)

	return &response
}
