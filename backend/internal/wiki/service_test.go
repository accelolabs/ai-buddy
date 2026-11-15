package wiki

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type syncFunc func(context.Context, []Document) error

func (f syncFunc) Sync(ctx context.Context, documents []Document) error {
	return f(ctx, documents)
}

func TestServiceReadFailure(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# Readable"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "missing"), filepath.Join(root, "z.md")); err != nil {
		t.Fatal(err)
	}
	repository := syncFunc(func(context.Context, []Document) error {
		t.Fatal("repository called after incomplete scan")
		return nil
	})
	for _, directory := range []string{root, filepath.Join(root, "missing")} {
		if err := NewService(repository, directory).Index(context.Background()); err == nil {
			t.Fatalf("Index(%q) succeeded despite read error", directory)
		}
	}
}

func TestServiceSyncError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("sync failed")
	called := false
	repository := syncFunc(func(gotCtx context.Context, documents []Document) error {
		called = true
		if gotCtx != ctx || len(documents) != 0 {
			t.Fatalf("Sync received context %v and documents %+v", gotCtx, documents)
		}
		return wantErr
	})
	err := NewService(repository, t.TempDir()).Index(ctx)
	if !called || !errors.Is(err, wantErr) {
		t.Fatalf("Index error = %v, called = %v; want wrapped sync error", err, called)
	}
}
