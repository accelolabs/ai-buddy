package buddy

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ai-buddy/internal/users"
)

type boundaryRepository struct {
	Repository
	calls int
}

func (r *boundaryRepository) Start(context.Context, string, string, string, string) (Run, bool, error) {
	r.calls++
	return Run{ID: "run"}, false, nil
}

func TestSendByteAndRuneBoundaries(t *testing.T) {
	repo := &boundaryRepository{}
	s := &Service{Repo: repo}
	for _, tc := range []struct {
		key, content string
		valid        bool
	}{
		{strings.Repeat("я", 100), strings.Repeat("я", 2000), true},
		{strings.Repeat("я", 101), "T", false}, {" ", "T", false},
		{"key", strings.Repeat("я", 2001), false}, {"key", " \t", false},
	} {
		before := repo.calls
		_, err := s.Send(context.Background(), users.User{ID: "i"}, "chat", tc.key, tc.content)
		if tc.valid {
			if err != nil || repo.calls != before+1 {
				t.Fatal(err, repo.calls)
			}
		} else if !errors.Is(err, ErrInvalid) || repo.calls != before {
			t.Fatal(err, repo.calls)
		}
	}
}
