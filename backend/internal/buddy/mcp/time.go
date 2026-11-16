package mcp

import (
	"context"
	"time"
	_ "time/tzdata" // Include timezone data in the minimal server image.

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type CurrentTimeArgs struct{}

type CurrentTimeResult struct {
	CurrentTime string `json:"current_time"`
	Timezone    string `json:"timezone"`
}

func (h *Handler) GetCurrentTime(
	_ context.Context,
	_ *sdk.CallToolRequest,
	_ CurrentTimeArgs,
) (*sdk.CallToolResult, CurrentTimeResult, error) {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return nil, CurrentTimeResult{}, err
	}

	return nil, CurrentTimeResult{
		CurrentTime: time.Now().In(location).Format(time.RFC3339),
		Timezone:    location.String(),
	}, nil
}
