// Package wiki indexes Markdown documents and searches the permitted corpus.
package wiki

import (
	"ai-buddy/internal/ollama"
	"context"
	"errors"
)

var (
	ErrUnavailable = ollama.ErrUnavailable
	ErrNotFound    = errors.New("document not found")
	ErrContext     = ollama.ErrContext
)

const SplitVersion = "goldmark-sections-runes-1200-150-v1"

type Document struct {
	ID       string
	Path     string
	Team     string
	Title    string
	Markdown string
	Hash     string
	Sections []Section
	Chunks   []Chunk
}

type Section struct {
	Title  string
	Anchor string
	Text   string
}

type Chunk struct {
	Section   int
	Text      string
	Embedding []float64
}

type Source struct {
	DocumentID string `json:"document_id"`
	Title      string `json:"document_title"`
	Section    string `json:"section_title"`
	Text       string `json:"text"`
	URL        string `json:"url"`
	Anchor     string `json:"-"`
}

type Stats struct {
	Added   int
	Changed int
	Deleted int
	Skipped int
}

type Embedder interface {
	Fingerprint(context.Context) (string, error)
	Embed(context.Context, string) ([]float64, error)
}
