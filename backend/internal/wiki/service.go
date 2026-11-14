package wiki

import (
	"context"
	"errors"
)

var ErrQuery = errors.New("query must contain 1 to 4000 characters")

type Store interface {
	List(context.Context, string) ([]Document, error)
	Get(context.Context, string, string) (Document, error)
	Search(context.Context, string, string, []float64, float64, string) ([]Result, error)
}

type Service struct {
	Store    Store
	Embedder Embedder
	Minimum  float64
}

func (s *Service) Search(ctx context.Context, team, query string) ([]Result, error) {
	fingerprint, err := s.Embedder.Fingerprint(ctx)
	if err != nil {
		return nil, ErrUnavailable
	}

	embedding, err := s.Embedder.Embed(ctx, "task: search result | query: "+query)
	if err != nil {
		return nil, ErrUnavailable
	}

	results, err := s.Store.Search(ctx, team, query, embedding, s.Minimum, fingerprint)
	if err != nil {
		return nil, ErrUnavailable
	}

	return results, nil
}
