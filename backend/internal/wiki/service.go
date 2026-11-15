package wiki

import (
	"context"

	"ai-buddy/internal/users"
)

type Repository interface {
	List(context.Context, string) ([]Document, error)
	Get(context.Context, string, string) (Document, error)
	Search(context.Context, string, string, []float64, float64, string) ([]Source, error)
}

type Service struct {
	Store    Repository
	Embedder Embedder
	Minimum  float64
	Profiles users.Reader
}

func (s *Service) Search(ctx context.Context, id, query string) ([]Source, error) {
	u, err := s.Profiles.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	fingerprint, err := indexFingerprint(ctx, s.Embedder)
	if err != nil {
		return nil, ErrUnavailable
	}

	embedding, err := s.Embedder.Embed(ctx, "task: search result | query: "+query)
	if err != nil {
		return nil, ErrUnavailable
	}

	results, err := s.Store.Search(ctx, u.Team, query, embedding, s.Minimum, fingerprint)
	if err != nil {
		return nil, ErrUnavailable
	}

	return results, nil
}
