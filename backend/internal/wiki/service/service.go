package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"

	// Project packages.
	"backend/internal/users"
	"backend/internal/wiki"
)

var (
	ErrNotFound = errors.New("article not found")
	ErrReadFile = errors.New("failed to read article")
)

type documentRepository interface {
	List(context.Context) ([]wiki.Document, error)
	Get(context.Context, string) (wiki.Document, error)
}

type Service struct {
	repository documentRepository
	root       string
}

func NewService(repository documentRepository, root string) *Service {
	return &Service{
		repository: repository,
		root:       root,
	}
}

func (s *Service) List(ctx context.Context, user users.User) ([]wiki.Document, error) {
	documents, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	values := make([]wiki.Document, 0, len(documents))

	for _, document := range documents {
		if document.Team == "" || document.Team == user.Team {
			values = append(values, document)
		}
	}

	return values, nil
}

func (s *Service) Get(ctx context.Context, user users.User, id string) ([]byte, error) {
	document, err := s.repository.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	if document.Team != "" && document.Team != user.Team {
		return nil, ErrNotFound
	}

	root, err := os.OpenRoot(s.root)
	if err != nil {
		return nil, fmt.Errorf("%w: open knowledge base: %w", ErrReadFile, err)
	}

	defer func() {
		_ = root.Close()
	}()

	content, err := root.ReadFile(document.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("%w: read wiki %q: %w", ErrReadFile, document.ID, err)
	}

	return content, nil
}
