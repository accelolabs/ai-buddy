package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handler) OpenSession(ctx context.Context) (*sdk.ClientSession, func(), error) {
	server := sdk.NewServer(&sdk.Implementation{
		Name:    "buddy",
		Version: "1",
	}, nil)
	h.Register(server)
	clientTransport, serverTransport := sdk.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("connect MCP server: %w", err)
	}

	client := sdk.NewClient(&sdk.Implementation{
		Name:    "buddy-agent",
		Version: "1",
	}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()

		return nil, nil, fmt.Errorf("connect MCP client: %w", err)
	}

	return session, func() {
		_ = session.Close()
		_ = serverSession.Close()
	}, nil
}
