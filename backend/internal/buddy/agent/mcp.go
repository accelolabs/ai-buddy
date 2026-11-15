package agent

import (
	"backend/internal/users"
	wikimcp "backend/internal/wiki/mcp"
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (a *Agent) openSession(ctx context.Context, user users.User) (*sdk.ClientSession, func(), error) {
	server := sdk.NewServer(&sdk.Implementation{Name: "buddy", Version: "1"}, nil)
	wikimcp.NewHandler(a.rag, user.Team).Register(server)
	clientTransport, serverTransport := sdk.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("connect MCP server: %w", err)
	}
	client := sdk.NewClient(&sdk.Implementation{Name: "buddy-agent", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		return nil, nil, fmt.Errorf("connect MCP client: %w", err)
	}
	return session, func() { _ = session.Close(); _ = serverSession.Close() }, nil
}
