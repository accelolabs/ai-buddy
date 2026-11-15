package indexer

import (
	"backend/internal/wiki"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type syncFunc func(context.Context, []wiki.Document, []wiki.Chunk) error

func (f syncFunc) Sync(ctx context.Context, documents []wiki.Document, chunks []wiki.Chunk) error {
	return f(ctx, documents, chunks)
}

type embedFunc func(context.Context, string) ([]float64, error)

func (f embedFunc) Embed(ctx context.Context, input string) ([]float64, error) {
	return f(ctx, input)
}

func TestIndexerReadFailure(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# Readable"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "missing"), filepath.Join(root, "z.md")); err != nil {
		t.Fatal(err)
	}
	repository := syncFunc(func(context.Context, []wiki.Document, []wiki.Chunk) error {
		t.Fatal("repository called after incomplete scan")
		return nil
	})
	embedding := embedFunc(func(context.Context, string) ([]float64, error) {
		t.Fatal("embedding requested after incomplete scan")
		return nil, nil
	})
	for _, directory := range []string{root, filepath.Join(root, "missing")} {
		if err := NewIndexer(repository, embedding, directory).Index(context.Background()); err == nil {
			t.Fatalf("Index(%q) succeeded despite read error", directory)
		}
	}
}

func TestIndexerSyncError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("sync failed")
	called := false
	repository := syncFunc(func(gotCtx context.Context, documents []wiki.Document, chunks []wiki.Chunk) error {
		called = true
		if gotCtx != ctx || len(documents) != 0 || len(chunks) != 0 {
			t.Fatalf("Sync received context %v and documents %+v", gotCtx, documents)
		}
		return wantErr
	})
	err := NewIndexer(repository, embedFunc(func(context.Context, string) ([]float64, error) {
		t.Fatal("embedding requested for empty corpus")
		return nil, nil
	}), t.TempDir()).Index(ctx)
	if !called || !errors.Is(err, wantErr) {
		t.Fatalf("Index error = %v, called = %v; want wrapped sync error", err, called)
	}
}

func TestIndexerEmbeddingFailure(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "guide.md"), []byte("# Первый\nТекст\n\n## Второй\nДетали"), 0o644); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("ollama failed")
	calls := 0
	ctx := context.Background()
	embedding := embedFunc(func(gotCtx context.Context, _ string) ([]float64, error) {
		if gotCtx != ctx {
			t.Fatal("embedding did not receive request context")
		}
		calls++
		if calls == 2 {
			return nil, wantErr
		}
		return []float64{0.5}, nil
	})
	repository := syncFunc(func(context.Context, []wiki.Document, []wiki.Chunk) error {
		t.Fatal("repository called after embedding failure")
		return nil
	})
	err := NewIndexer(repository, embedding, root).Index(ctx)
	if calls != 2 || !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "/1") {
		t.Fatalf("Index error=%v; embedding calls=%d", err, calls)
	}
}

func TestIndexerCancelled(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "guide.md"), []byte("# Статья\nТекст"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	embedding := embedFunc(func(context.Context, string) ([]float64, error) {
		t.Fatal("embedding called with cancelled context")
		return nil, nil
	})
	repository := syncFunc(func(context.Context, []wiki.Document, []wiki.Chunk) error {
		t.Fatal("repository called with cancelled context")
		return nil
	})
	if err := NewIndexer(repository, embedding, root).Index(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Index error = %v", err)
	}
}
