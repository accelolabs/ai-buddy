package wiki

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
)

type documentSyncer interface {
	Sync(context.Context, []Document) error
}

type Service struct {
	repository documentSyncer
	root       string
}

func NewService(repository documentSyncer, root string) *Service {
	return &Service{repository: repository, root: root}
}

func (s *Service) Index(ctx context.Context) error {
	files := os.DirFS(s.root)
	documents := make([]Document, 0)
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

func parseDocument(name, content string) Document {
	document := Document{
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
