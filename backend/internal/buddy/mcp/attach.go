package mcp

import (
	"context"
	"fmt"

	// Project packages.
	"backend/internal/buddy"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type key struct {
	documentID string
	ordinal    int
}

type AttachArgs struct {
	DocumentID string `json:"document_id"`
	Ordinal    int    `json:"ordinal"`
}

type AttachResult struct {
	Attached bool `json:"attached"`
}

func (h *Handler) Attach(
	_ context.Context,
	_ *sdk.CallToolRequest,
	args AttachArgs,
) (*sdk.CallToolResult, AttachResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	selected := key{args.DocumentID, args.Ordinal}
	_, ok := h.found[selected]
	if !ok {
		return nil, AttachResult{}, fmt.Errorf("source was not returned by search in this run")
	}

	h.selected = &selected

	return nil, AttachResult{Attached: true}, nil
}

func (h *Handler) Source() *buddy.Source {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.selected == nil {
		return nil
	}

	source := h.found[*h.selected]

	return &source
}
