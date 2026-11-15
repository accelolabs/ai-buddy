package mcp

import (
	"backend/internal/wiki"
	"context"
	"encoding/json"
	"errors"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeRAG struct {
	chunks      []wiki.Chunk
	err         error
	query, team string
}

func (f *fakeRAG) Search(_ context.Context, query, team string) ([]wiki.Chunk, error) {
	f.query, f.team = query, team
	return f.chunks, f.err
}

func TestSearch(t *testing.T) {
	ctx := t.Context()
	source := &fakeRAG{chunks: []wiki.Chunk{{DocumentID: "doc", Ordinal: 2, Anchor: "setup", Text: "Setup"}}}
	h := NewHandler(source, "backend")
	_, found, err := h.Search(ctx, nil, SearchArgs{Query: "question"})
	if err != nil || len(found.Chunks) != 1 || source.query != "question" || source.team != "backend" {
		t.Fatalf("search: %+v, %v", found, err)
	}
	want := Chunk{DocumentID: "doc", Ordinal: 2, Anchor: "setup", Text: "Setup"}
	if found.Chunks[0] != want {
		t.Fatalf("chunk: %+v", found.Chunks[0])
	}

	source.chunks = nil
	_, empty, err := h.Search(ctx, nil, SearchArgs{Query: "empty"})
	if err != nil || empty.Chunks == nil || len(empty.Chunks) != 0 {
		t.Fatalf("empty: %+v, %v", empty, err)
	}
	source.chunks = []wiki.Chunk{{DocumentID: "failed", Text: "Hidden"}}
	source.err = errors.New("search failed")
	if _, _, err := h.Search(ctx, nil, SearchArgs{Query: "failure"}); !errors.Is(err, source.err) {
		t.Fatal(err)
	}
}

func TestMCPTools(t *testing.T) {
	ctx := t.Context()
	server := sdk.NewServer(&sdk.Implementation{Name: "test", Version: "1"}, nil)
	NewHandler(&fakeRAG{chunks: []wiki.Chunk{{DocumentID: "doc", Text: "Text"}}}, "backend").Register(server)
	ct, st := sdk.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ss.Close() }()
	client := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	listed, err := session.ListTools(ctx, nil)
	if err != nil || len(listed.Tools) != 1 || listed.Tools[0].Name != "search_documents" {
		t.Fatalf("tools: %v, %v", listed, err)
	}
	call := func(name string, args map[string]any, wantError bool) *sdk.CallToolResult {
		t.Helper()
		result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		if result.IsError != wantError {
			t.Fatalf("%s: %+v", name, result)
		}
		return result
	}
	call("search_documents", map[string]any{}, true)
	result := call("search_documents", map[string]any{"query": "question"}, false)
	encoded, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var output SearchResult
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatal(err)
	}
	if len(output.Chunks) != 1 || output.Chunks[0].DocumentID != "doc" || output.Chunks[0].Text != "Text" {
		t.Fatalf("search: %s", encoded)
	}
}
