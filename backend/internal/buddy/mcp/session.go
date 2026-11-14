// Package mcp hosts a private MCP server for each trusted run.
package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/platform/cleanup"
)

type Session struct {
	Client *sdk.ClientSession
	server *sdk.ServerSession
}

func (s *Session) Close() {
	cleanup.Close(s.Client)
	cleanup.Close(s.server)
}

func (t *Tools) Open(ctx context.Context, u auth.User, run buddy.Run, request string) (*Session, error) {
	server := t.register(u, run, request)
	ct, st := sdk.NewInMemoryTransports()
	ss, e := server.Connect(ctx, st, nil)
	if e != nil {
		return nil, e
	}
	client := sdk.NewClient(&sdk.Implementation{Name: "buddy-agent", Version: "1"}, nil)
	cs, e := client.Connect(ctx, ct, nil)
	if e != nil {
		cleanup.Close(ss)
		return nil, e
	}
	return &Session{cs, ss}, nil
}
