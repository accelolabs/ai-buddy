package wiki

import (
	"context"
	"errors"
	"fmt"
)

func embedChunk(
	ctx context.Context,
	embedder Embedder,
	title string,
	chunk Chunk,
) ([]Chunk, error) {
	indexEmbedding, err := embedder.Embed(
		ctx,
		fmt.Sprintf("title: %s | text: %s", title, chunk.Text),
	)
	if errors.Is(err, ErrContext) {
		characters := []rune(chunk.Text)
		if len(characters) < 2 {
			return nil, err
		}

		mid := len(characters) / 2

		left, err := embedChunk(
			ctx,
			embedder,
			title,
			Chunk{Section: chunk.Section, Text: string(characters[:mid])},
		)
		if err != nil {
			return nil, err
		}

		right, err := embedChunk(
			ctx,
			embedder,
			title,
			Chunk{Section: chunk.Section, Text: string(characters[mid:])},
		)

		return append(left, right...), err
	}

	if err != nil {
		return nil, err
	}

	chunk.Embedding = indexEmbedding

	return []Chunk{chunk}, nil
}
