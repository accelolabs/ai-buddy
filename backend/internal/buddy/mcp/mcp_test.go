package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	// Project packages.
	"backend/internal/users"
	"backend/internal/wiki"
	"backend/internal/wiki/rag"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeRAG struct {
	chunks      []rag.Match
	err         error
	query, team string
}

func (f *fakeRAG) Search(_ context.Context, query, team string) ([]rag.Match, error) {
	f.query, f.team = query, team

	return f.chunks, f.err
}

func TestSearch(t *testing.T) {
	ctx := t.Context()
	source := &fakeRAG{
		chunks: []rag.Match{{
			Document: wiki.Document{Title: "Title"},
			Chunk: wiki.Chunk{
				DocumentID: "doc",
				Ordinal:    2,
				Anchor:     "setup",
				Text:       "Setup",
			},
		}},
	}

	h := NewHandler(source, users.User{Team: "backend"}, nil, nil)
	_, found, err := h.Search(ctx, nil, SearchArgs{Query: "question"})
	if err != nil || len(found.Chunks) != 1 || source.query != "question" || source.team != "backend" {
		t.Fatalf("search: %+v, %v", found, err)
	}

	want := Chunk{
		Title:      "Title",
		DocumentID: "doc",
		Ordinal:    2,
		Anchor:     "setup",
		Text:       "Setup",
	}

	if found.Chunks[0] != want {
		t.Fatalf("chunk: %+v", found.Chunks[0])
	}

	source.chunks = nil
	_, empty, err := h.Search(ctx, nil, SearchArgs{Query: "empty"})
	if err != nil || empty.Chunks == nil || len(empty.Chunks) != 0 {
		t.Fatalf("empty: %+v, %v", empty, err)
	}

	source.chunks = []rag.Match{{Chunk: wiki.Chunk{
		DocumentID: "failed",
		Text:       "Hidden",
	}}}
	source.err = errors.New("search failed")
	if _, _, err := h.Search(ctx, nil, SearchArgs{Query: "failure"}); !errors.Is(err, source.err) {
		t.Fatal(err)
	}
}

func TestMCPTools(t *testing.T) {
	ctx := t.Context()
	server := sdk.NewServer(&sdk.Implementation{
		Name:    "test",
		Version: "1",
	}, nil)
	NewHandler(
		&fakeRAG{chunks: []rag.Match{{Chunk: wiki.Chunk{
			DocumentID: "doc",
			Text:       "Text",
		}}}},
		users.User{
			Team: "backend",
		},
		nil,
		nil,
	).Register(server)
	ct, st := sdk.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = ss.Close()
	}()

	client := sdk.NewClient(&sdk.Implementation{
		Name:    "test-client",
		Version: "1",
	}, nil)
	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = session.Close()
	}()

	listed, err := session.ListTools(ctx, nil)
	if err != nil || len(listed.Tools) != 2 || listed.Tools[0].Name != "attach_source" ||
		listed.Tools[1].Name != "search_documents" {
		t.Fatalf("tools: %v, %v", listed, err)
	}

	call := func(name string, args map[string]any, wantError bool) *sdk.CallToolResult {
		t.Helper()
		result, err := session.CallTool(ctx, &sdk.CallToolParams{
			Name:      name,
			Arguments: args,
		})
		if err != nil {
			t.Fatal(err)
		}

		if result.IsError != wantError {
			t.Fatalf("%s: %+v", name, result)
		}

		return result
	}

	call("search_documents", map[string]any{}, true)
	call("attach_source", map[string]any{
		"document_id": "doc",
		"ordinal":     0,
	}, true)
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

	call("attach_source", map[string]any{"document_id": "doc"}, true)
	selected := call("attach_source", map[string]any{
		"document_id": "doc",
		"ordinal":     0,
	}, false)
	encoded, err = json.Marshal(selected.StructuredContent)
	if err != nil || string(encoded) != `{"attached":true}` {
		t.Fatalf("attach: %s %v", encoded, err)
	}
}

func TestAttachSessionResults(t *testing.T) {
	ctx := t.Context()
	search := &fakeRAG{
		chunks: []rag.Match{{
			Document: wiki.Document{Title: "Title"},
			Chunk: wiki.Chunk{
				DocumentID: "a",
				Ordinal:    2,
				Anchor:     "экватор",
				Text:       "Old",
			},
		}},
	}

	h := NewHandler(search, users.User{Team: "backend"}, nil, nil)
	attach := func(id string, ordinal int, wantError bool) {
		t.Helper()
		_, result, err := h.Attach(ctx, nil, AttachArgs{
			DocumentID: id,
			Ordinal:    ordinal,
		})
		if (err != nil) != wantError || result.Attached == wantError {
			t.Fatalf("attach: %+v, %v", result, err)
		}
	}

	attach("a", 2, true)
	if _, _, err := h.Search(ctx, nil, SearchArgs{Query: "one"}); err != nil {
		t.Fatal(err)
	}

	attach("a", 2, false)
	if got := h.Source(); got.Title != "Title" || got.Anchor != "экватор" || got.Quote != "Old" {
		t.Fatal(got)
	}

	search.chunks[0].Chunk.Text = "Updated"
	if _, _, err := h.Search(ctx, nil, SearchArgs{Query: "two"}); err != nil {
		t.Fatal(err)
	}

	if h.Source().Quote != "Updated" {
		t.Fatal(h.Source())
	}

	search.chunks = []rag.Match{{Chunk: wiki.Chunk{
		DocumentID: "b",
		Ordinal:    0,
		Text:       "Other",
	}}}
	if _, _, err := h.Search(ctx, nil, SearchArgs{Query: "three"}); err != nil {
		t.Fatal(err)
	}

	attach("a", 2, false) // Previous successful searches remain selectable.
	attach("b", 0, false)
	attach("a", 3, true)
	if h.Source().DocumentID != "b" {
		t.Fatal("failed attach changed source")
	}

	search.err = errors.New("failed search")
	search.chunks = []rag.Match{{Chunk: wiki.Chunk{DocumentID: "hidden"}}}
	if _, _, err := h.Search(ctx, nil, SearchArgs{Query: "fail"}); err == nil {
		t.Fatal("expected search error")
	}

	attach("hidden", 0, true)
	other := NewHandler(search, users.User{Team: "platform"}, nil, nil)
	if _, _, err := other.Attach(ctx, nil, AttachArgs{DocumentID: "b"}); err == nil || other.Source() != nil {
		t.Fatal("results leaked across runs")
	}

	snapshot := h.Source()
	snapshot.Quote = "mutated"
	if h.Source().Quote != "Other" {
		t.Fatal("source exposes mutable state")
	}
}
