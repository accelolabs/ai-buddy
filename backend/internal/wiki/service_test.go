package wiki

import (
	"context"
	"errors"
	"testing"
)

type fakeStore struct {
	err             error
	team, query, fp string
}

func (f *fakeStore) List(context.Context, string) ([]Document, error)      { return nil, f.err }
func (f *fakeStore) Get(context.Context, string, string) (Document, error) { return Document{}, f.err }
func (f *fakeStore) Search(_ context.Context, team, query string, v []float64, min float64, fp string) ([]Result, error) {
	f.team, f.query, f.fp = team, query, fp
	return []Result{{DocumentID: "a"}}, f.err
}

func TestService(t *testing.T) {
	f := &fakeEmbedder{fp: "model"}
	store := &fakeStore{}
	s := Service{store, f, 0.35}
	ctx := context.Background()

	found, err := s.Search(ctx, "backend", "текст")
	if err != nil || len(found) != 1 || store.team != "backend" || store.query != "текст" || store.fp != "model" {
		t.Fatal(found, err, store)
	}
	f.err = ErrUnavailable
	if _, err := s.Search(ctx, "backend", "text"); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	f.err = nil
	store.err = errors.New("db")
	if _, err := s.Search(ctx, "backend", "text"); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}
