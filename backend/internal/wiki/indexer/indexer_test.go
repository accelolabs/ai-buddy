package indexer

import (
	"backend/internal/wiki"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type syncFunc func(context.Context, []wiki.Document, []wiki.Chunk) error

func (f syncFunc) Sync(ctx context.Context, documents []wiki.Document, chunks []wiki.Chunk) error {
	return f(ctx, documents, chunks)
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
	for _, directory := range []string{root, filepath.Join(root, "missing")} {
		if err := NewIndexer(repository, directory).Index(context.Background()); err == nil {
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
	err := NewIndexer(repository, t.TempDir()).Index(ctx)
	if !called || !errors.Is(err, wantErr) {
		t.Fatalf("Index error = %v, called = %v; want wrapped sync error", err, called)
	}
}
