package indexer

import (
	"context"
	"fmt"

	"ai-buddy/internal/wiki"
)

func embedChunk(ctx context.Context, embedder wiki.Embedder, title string, chunk wiki.Chunk) ([]wiki.Chunk, error) {
	embedding, err := embedder.Embed(ctx, fmt.Sprintf("title: %s | text: %s", title, chunk.Text))
	if err == wiki.ErrContext {
		characters := []rune(chunk.Text)
		if len(characters) < 2 {
			return nil, err
		}
		mid := len(characters) / 2
		left, err := embedChunk(ctx, embedder, title, wiki.Chunk{Section: chunk.Section, Text: string(characters[:mid])})
		if err != nil {
			return nil, err
		}
		right, err := embedChunk(ctx, embedder, title, wiki.Chunk{Section: chunk.Section, Text: string(characters[mid:])})
		return append(left, right...), err
	}
	if err != nil {
		return nil, err
	}

	chunk.Embedding = embedding
	return []wiki.Chunk{chunk}, nil
}
