package wiki

import (
	"context"
	"errors"
)

var ErrUnavailable = errors.New("search dependency unavailable")
var ErrNotFound = errors.New("document not found")
var ErrContext = errors.New("embedding context exceeded")

const SplitVersion = "goldmark-sections-runes-1200-150-v1"
const Model = "embeddinggemma:300m"

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

type Result struct {
	DocumentID    string
	DocumentTitle string
	SectionTitle  string
	Anchor        string
	Text          string
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
