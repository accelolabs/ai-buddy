package indexer

import (
	"backend/internal/wiki"
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
)

type documentSyncer interface {
	Sync(context.Context, []wiki.Document) error
}

type Indexer struct {
	repository documentSyncer
	root       string
}

func NewIndexer(repository documentSyncer, root string) *Indexer {
	return &Indexer{repository: repository, root: root}
}

func (s *Indexer) Index(ctx context.Context) error {
	files := os.DirFS(s.root)
	documents := make([]wiki.Document, 0)
	err := fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || path.Ext(name) != ".md" {
			return nil
		}

		content, err := fs.ReadFile(files, name)
		if err != nil {
			return err
		}
		documents = append(documents, parseDocument(name, string(content)))
		return nil
	})
	if err != nil {
		return fmt.Errorf("read knowledge base: %w", err)
	}
	if err := s.repository.Sync(ctx, documents); err != nil {
		return fmt.Errorf("index wiki: %w", err)
	}

	return nil
}

func parseDocument(name, content string) wiki.Document {
	document := wiki.Document{
		ID:    fmt.Sprintf("%x", sha256.Sum256([]byte(name))),
		Path:  name,
		Title: strings.TrimSuffix(path.Base(name), ".md"),
	}
	if team, _, nested := strings.Cut(name, "/"); nested {
		document.Team = team
	}
	for line := range strings.SplitSeq(content, "\n") {
		if title, ok := strings.CutPrefix(line, "# "); ok {
			document.Title = strings.TrimSpace(title)
			break
		}
	}

	return document
}
