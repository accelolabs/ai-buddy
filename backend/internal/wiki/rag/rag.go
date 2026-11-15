package rag

import (
	"backend/internal/wiki"
	"context"
	"fmt"
	"strings"
)

type repository interface {
	List(context.Context) ([]wiki.Document, error)
	SearchChunks(context.Context, []float64, []string, int) ([]wiki.Chunk, error)
}

type embedder interface {
	Embed(context.Context, string) ([]float64, error)
}

type RAG struct {
	repository repository
	embedder   embedder
	chunkCount int
}

func NewRAG(repository repository, embedder embedder, chunkCount int) *RAG {
	return &RAG{repository: repository, embedder: embedder, chunkCount: chunkCount}
}

func (r *RAG) Search(ctx context.Context, query, team string) ([]wiki.Chunk, error) {
	if r.chunkCount <= 0 {
		return nil, fmt.Errorf("rag chunk count must be positive")
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("rag query must not be empty")
	}
	documents, err := r.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list rag documents: %w", err)
	}
	ids := make([]string, 0, len(documents))
	for _, document := range documents {
		if document.Team == "" || document.Team == team {
			ids = append(ids, document.ID)
		}
	}
	if len(ids) == 0 {
		return []wiki.Chunk{}, nil
	}
	embedding, err := r.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed rag query: %w", err)
	}
	chunks, err := r.repository.SearchChunks(ctx, embedding, ids, r.chunkCount)
	if err != nil {
		return nil, fmt.Errorf("search rag chunks: %w", err)
	}
	return chunks, nil
}
