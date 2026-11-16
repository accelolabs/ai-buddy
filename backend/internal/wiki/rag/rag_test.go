package rag

import (
	"context"
	"errors"
	"reflect"
	"testing"

	// Project packages.
	"backend/internal/wiki"
)

type fakeRepository struct {
	documents []wiki.Document
	chunks    []wiki.Chunk
	listErr   error
	searchErr error
	ids       []string
	embedding []float64
	limit     int
}

func (f *fakeRepository) List(context.Context) ([]wiki.Document, error) {
	return f.documents, f.listErr
}

func (f *fakeRepository) SearchChunks(
	_ context.Context,
	embedding []float64,
	ids []string,
	limit int,
) ([]wiki.Chunk, error) {
	f.embedding, f.ids, f.limit = embedding, ids, limit

	return f.chunks, f.searchErr
}

type fakeEmbedder struct {
	query string
	err   error
}

func (f *fakeEmbedder) Embed(_ context.Context, query string) ([]float64, error) {
	f.query = query

	return []float64{1, 2}, f.err
}

func TestSearch(t *testing.T) {
	for _, count := range []int{1, 3, 5} {
		repo := &fakeRepository{
			documents: []wiki.Document{{ID: "common"}, {
				ID:    "own",
				Title: "Title",
				Team:  "backend",
			}, {
				ID:   "foreign",
				Team: "platform",
			}},
			chunks: []wiki.Chunk{{
				DocumentID: "own",
				Ordinal:    2,
				Anchor:     "setup",
				Text:       "Setup",
			}},
		}

		embed := &fakeEmbedder{}
		got, err := NewRAG(repo, embed, count).Search(t.Context(), "Как начать?", "backend")
		if err != nil || !reflect.DeepEqual(got, []Match{{
			Document: repo.documents[1],
			Chunk:    repo.chunks[0],
		}}) {
			t.Fatalf("Search = %v, %v", got, err)
		}

		if !reflect.DeepEqual(repo.ids, []string{"common", "own"}) || repo.limit != count ||
			!reflect.DeepEqual(
				repo.embedding,
				[]float64{1, 2},
			) ||
			embed.query != "Как начать?" {
			t.Fatalf("wrong search inputs: %+v, %+v", repo, embed)
		}
	}
}

func TestSearchFailures(t *testing.T) {
	failure := errors.New("failure")

	for _, stage := range []string{"list", "embed", "search"} {
		t.Run(stage, func(t *testing.T) {
			repo := &fakeRepository{documents: []wiki.Document{{ID: "common"}}}
			embed := &fakeEmbedder{}

			switch stage {
			case "list":
				repo.listErr = failure
			case "embed":
				embed.err = failure
			case "search":
				repo.searchErr = failure
			}

			_, err := NewRAG(repo, embed, 3).Search(t.Context(), "query", "backend")
			if !errors.Is(err, failure) {
				t.Fatalf("error = %v", err)
			}

			if stage != "search" && repo.ids != nil {
				t.Fatal("search called after error")
			}
		})
	}
}

func TestSearchEmptyAndInvalid(t *testing.T) {
	repo := &fakeRepository{documents: []wiki.Document{{
		ID:   "foreign",
		Team: "platform",
	}}}
	embed := &fakeEmbedder{}
	got, err := NewRAG(repo, embed, 3).Search(t.Context(), "query", "backend")
	if err != nil || got == nil || len(got) != 0 || embed.query != "" || repo.ids != nil {
		t.Fatalf("empty corpus = %v, %v", got, err)
	}

	for _, count := range []int{0, -1} {
		if _, err := NewRAG(repo, embed, count).Search(t.Context(), "query", "backend"); err == nil {
			t.Fatal("invalid count accepted")
		}
	}

	if _, err := NewRAG(repo, embed, 3).Search(t.Context(), " \n", "backend"); err == nil {
		t.Fatal("empty query accepted")
	}
}
