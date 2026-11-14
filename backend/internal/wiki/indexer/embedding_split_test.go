package indexer

import (
	"context"
	"testing"

	"ai-buddy/internal/wiki"
)

func TestOverflowSplit(t *testing.T) {
	f := &fakeEmbedder{limit: 3}
	chunks, err := embedChunk(context.Background(), f, "Title", wiki.Chunk{Section: 2, Text: "абвгдежз"})
	if err != nil || len(chunks) != 4 {
		t.Fatal(chunks, err)
	}
	for _, c := range chunks {
		if c.Section != 2 || len(c.Embedding) != 768 {
			t.Fatal(c)
		}
	}
	f.err = wiki.ErrContext
	if _, err := embedChunk(context.Background(), f, "t", wiki.Chunk{Text: "a"}); err != wiki.ErrContext {
		t.Fatal(err)
	}
	f.err = wiki.ErrUnavailable
	if _, err := embedChunk(context.Background(), f, "t", wiki.Chunk{Text: "a"}); err == nil {
		t.Fatal("failure")
	}
}
