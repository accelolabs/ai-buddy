package wiki

import (
	"context"
	"errors"
)

type Changes struct {
	Documents   []Document
	Deleted     []string
	Fingerprint string
}

func Run(ctx context.Context, store *Store, root string, embedder Embedder, full bool) (Stats, error) {
	stats := Stats{}
	session, err := store.Acquire(ctx)
	if err != nil {
		return stats, err
	}
	defer session.Close()

	fingerprint, err := indexFingerprint(ctx, embedder)
	if err != nil {
		return stats, err
	}
	stored, err := session.Fingerprint(ctx)
	if err != nil {
		return stats, err
	}
	if stored != "" && stored != fingerprint && !full {
		return stats, errors.New("embedding fingerprint changed; run with --full")
	}

	documents, err := ReadCorpus(root)
	if err != nil {
		return stats, err
	}
	hashes, err := session.Hashes(ctx)
	if err != nil {
		return stats, err
	}

	changes, stats, err := prepare(ctx, documents, hashes, embedder, full)
	if err != nil {
		return stats, err
	}
	changes.Fingerprint = fingerprint

	// A replacement during preparation must not publish mixed embeddings.
	final, err := indexFingerprint(ctx, embedder)
	if err != nil {
		return stats, err
	}
	if final != fingerprint {
		return stats, errors.New("embedding model changed during indexing")
	}

	return stats, session.Publish(ctx, changes)
}

func prepare(ctx context.Context, documents []Document, hashes map[string]string, embedder Embedder, full bool) (Changes, Stats, error) {
	changes := Changes{}
	stats := Stats{}

	for _, document := range documents {
		hash, exists := hashes[document.ID]
		delete(hashes, document.ID)

		if exists && hash == document.Hash && !full {
			stats.Skipped++
			continue
		}
		if exists {
			stats.Changed++
		} else {
			stats.Added++
		}

		chunks := []Chunk{}
		for _, chunk := range document.Chunks {
			parts, err := embedChunk(ctx, embedder, document.Title, chunk)
			if err != nil {
				return changes, stats, err
			}
			chunks = append(chunks, parts...)
		}
		document.Chunks = chunks
		changes.Documents = append(changes.Documents, document)
	}

	for id := range hashes {
		changes.Deleted = append(changes.Deleted, id)
	}
	stats.Deleted = len(changes.Deleted)

	return changes, stats, nil
}

func indexFingerprint(ctx context.Context, embedder Embedder) (string, error) {
	fingerprint, err := embedder.Fingerprint(ctx)
	if err != nil {
		return "", err
	}
	return fingerprint + ":" + SplitVersion, nil
}
