package indexer

import (
	"context"
	"errors"

	"ai-buddy/internal/wiki"
)

type Store interface {
	Acquire(context.Context) (Session, error)
}

// Session holds the exclusive update lock until Close, including preparation.
type Session interface {
	Fingerprint(context.Context) (string, error)
	Hashes(context.Context) (map[string]string, error)
	Publish(context.Context, Changes) error
	Close()
}

type Changes struct {
	Documents   []wiki.Document
	Deleted     []string
	Fingerprint string
}

func Run(ctx context.Context, store Store, root string, embedder wiki.Embedder, full bool) (wiki.Stats, error) {
	stats := wiki.Stats{}
	session, err := store.Acquire(ctx)
	if err != nil {
		return stats, err
	}
	defer session.Close()

	fingerprint, err := embedder.Fingerprint(ctx)
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
	final, err := embedder.Fingerprint(ctx)
	if err != nil {
		return stats, err
	}
	if final != fingerprint {
		return stats, errors.New("embedding model changed during indexing")
	}

	return stats, session.Publish(ctx, changes)
}

func prepare(ctx context.Context, documents []wiki.Document, hashes map[string]string, embedder wiki.Embedder, full bool) (Changes, wiki.Stats, error) {
	changes := Changes{}
	stats := wiki.Stats{}

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

		chunks := []wiki.Chunk{}
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
