package seed

import (
	"context"
	"errors"
	"testing"
)

func TestRun(t *testing.T) {
	if err := Run(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(Run(ctx, nil), context.Canceled) {
		t.Fatal("cancellation ignored")
	}
}
