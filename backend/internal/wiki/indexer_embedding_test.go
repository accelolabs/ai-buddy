package wiki

import (
	"context"
	"strings"
)

func indexEmbedding() []float64 { v := make([]float64, 768); v[0] = 1; return v }

type indexEmbedder struct {
	fp    string
	err   error
	calls int
	limit int
	hook  func()
}

func (f *indexEmbedder) Fingerprint(context.Context) (string, error) { return f.fp, f.err }
func (f *indexEmbedder) Embed(_ context.Context, s string) ([]float64, error) {
	f.calls++
	if f.hook != nil {
		f.hook()
	}
	if f.err != nil {
		return nil, f.err
	}
	_, body, _ := strings.Cut(s, " | text: ")
	if f.limit > 0 && len([]rune(body)) > f.limit {
		return nil, ErrContext
	}
	return indexEmbedding(), nil
}
