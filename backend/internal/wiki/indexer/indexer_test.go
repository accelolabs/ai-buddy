package indexer

import (
	"context"
	"testing"
)

type memoryStore struct {
	fingerprint  string
	hashes       map[string]string
	changes      Changes
	publications int
	closed       bool
}

func (s *memoryStore) Acquire(context.Context) (Session, error) {
	s.closed = false
	return s, nil
}

func (s *memoryStore) Fingerprint(context.Context) (string, error) {
	return s.fingerprint, nil
}

func (s *memoryStore) Hashes(context.Context) (map[string]string, error) {
	// Preparation owns its working copy; failed runs must not mutate stored hashes.
	hashes := map[string]string{}
	for id, hash := range s.hashes {
		hashes[id] = hash
	}
	return hashes, nil
}

func (s *memoryStore) Publish(_ context.Context, changes Changes) error {
	s.changes = changes
	s.publications++
	return nil
}

func (s *memoryStore) Close() {
	s.closed = true
}

func TestPrepareAndPublish(t *testing.T) {
	root := corpus(t, map[string]string{"backend/a.md": "# A\n\nтекст"})
	document := Parse("backend/a.md", "# A\n\nтекст")
	ctx := context.Background()

	for _, mode := range []string{"added", "unchanged", "deleted", "full", "embedding failure", "fingerprint mismatch"} {
		t.Run(mode, func(t *testing.T) {
			store := &memoryStore{fingerprint: "fp", hashes: map[string]string{}}
			embedder := &fakeEmbedder{fp: "fp"}
			switch mode {
			case "unchanged", "full":
				store.hashes[document.ID] = document.Hash
			case "deleted":
				store.hashes["removed"] = "old hash"
			case "embedding failure":
				embedder.hook = func() { embedder.err = context.DeadlineExceeded }
			case "fingerprint mismatch":
				store.fingerprint = "other model"
			}

			stats, err := Run(ctx, store, root, embedder, mode == "full")
			if !store.closed {
				t.Fatal("update session was not released")
			}
			if mode == "embedding failure" || mode == "fingerprint mismatch" {
				if err == nil || store.publications != 0 {
					t.Fatal("failed preparation published an index", err)
				}
				return
			}
			if err != nil || store.publications != 1 || store.changes.Fingerprint != "fp" {
				t.Fatal(stats, err, store.changes)
			}
			if mode == "unchanged" {
				if stats.Skipped != 1 || embedder.calls != 0 || len(store.changes.Documents) != 0 {
					t.Fatal(stats, embedder.calls)
				}
				return
			}

			if len(store.changes.Documents) != 1 || len(store.changes.Documents[0].Chunks[0].Embedding) != 768 {
				t.Fatal("publication did not receive prepared embeddings")
			}
			if mode == "deleted" && (stats.Deleted != 1 || len(store.changes.Deleted) != 1) {
				t.Fatal(stats, store.changes)
			}
			if mode == "full" && stats.Changed != 1 {
				t.Fatal(stats)
			}
		})
	}
}
