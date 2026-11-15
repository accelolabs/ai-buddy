package wiki

import (
	"context"
	"testing"
)

func TestOverflowSplit(t *testing.T) {
	f := &indexEmbedder{limit: 3}
	chunks, err := embedChunk(context.Background(), f, "Title", Chunk{Section: 2, Text: "абвгдежз"})
	if err != nil || len(chunks) != 4 {
		t.Fatal(chunks, err)
	}
	for _, c := range chunks {
		if c.Section != 2 || len(c.Embedding) != 768 {
			t.Fatal(c)
		}
	}
	f.err = ErrContext
	if _, err := embedChunk(context.Background(), f, "t", Chunk{Text: "a"}); err != ErrContext {
		t.Fatal(err)
	}
	f.err = ErrUnavailable
	if _, err := embedChunk(context.Background(), f, "t", Chunk{Text: "a"}); err == nil {
		t.Fatal("failure")
	}
}
