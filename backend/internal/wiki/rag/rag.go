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

type Match struct {
	Document wiki.Document
	Chunk    wiki.Chunk
}

func NewRAG(repository repository, embedder embedder, chunkCount int) *RAG {
	return &RAG{repository: repository, embedder: embedder, chunkCount: chunkCount}
}

func (r *RAG) Search(ctx context.Context, query, team string) ([]Match, error) {
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
	byID := make(map[string]wiki.Document, len(documents))
	for _, document := range documents {
		if document.Team == "" || document.Team == team {
			ids = append(ids, document.ID)
			byID[document.ID] = document
		}
	}
	if len(ids) == 0 {
		return []Match{}, nil
	}
	embedding, err := r.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed rag query: %w", err)
	}
	chunks, err := r.repository.SearchChunks(ctx, embedding, ids, r.chunkCount)
	if err != nil {
		return nil, fmt.Errorf("search rag chunks: %w", err)
	}
	matches := make([]Match, 0, len(chunks))
	for _, chunk := range chunks {
		matches = append(matches, Match{Document: byID[chunk.DocumentID], Chunk: chunk})
	}
	return matches, nil
}
