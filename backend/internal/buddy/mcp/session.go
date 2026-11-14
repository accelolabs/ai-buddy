// Package mcp hosts a private MCP server for each trusted run.
package mcp

import (
	"context"
	"log/slog"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/platform/auth"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Session struct {
	Client *sdk.ClientSession
	server *sdk.ServerSession
}

func (s *Session) Close() {
	if err := s.Client.Close(); err != nil {
		slog.Warn("resource cleanup failed", "error", err)
	}
	if err := s.server.Close(); err != nil {
		slog.Warn("resource cleanup failed", "error", err)
	}
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
		if err := ss.Close(); err != nil {
			slog.Warn("resource cleanup failed", "error", err)
		}
		return nil, e
	}
	return &Session{cs, ss}, nil
}
