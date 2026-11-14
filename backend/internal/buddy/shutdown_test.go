package buddy

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"ai-buddy/internal/platform/auth"
)

type blockedExecutor struct{ entered, release chan struct{} }

func (e blockedExecutor) Execute(ctx context.Context, _ auth.User, _ Action) (json.RawMessage, error) {
	close(e.entered)
	select {
	case <-e.release:
		return json.RawMessage(`{"id":"created"}`), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type blockedRunner struct{ entered chan struct{} }

func (r blockedRunner) Generate(ctx context.Context, _ auth.User, _ Run) error {
	close(r.entered)
	<-ctx.Done()
	return ctx.Err()
}
func TestCloseWaitsForGenerationAndConfirmation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered, release := make(chan struct{}), make(chan struct{})
	generation := make(chan struct{})
	repo := &repoStub{fresh: true, execute: true, finished: make(chan string, 1)}
	s := &Service{Repo: repo, Runner: blockedRunner{generation}, Executor: blockedExecutor{entered, release}, Context: ctx}
	if _, err := s.Send(ctx, auth.User{}, "chat", "key", "hello"); err != nil {
		t.Fatal(err)
	}
	<-generation
	confirmed := make(chan error, 1)
	request, cancelRequest := context.WithCancel(context.Background())
	go func() { _, err := s.Confirm(request, auth.User{}, "action", nil); confirmed <- err }()
	<-entered
	cancelRequest() // A lost browser connection must not cancel the downstream write.
	cancel()        // Stop the model while the independently timed confirmation continues.
	closed := make(chan struct{})
	go func() { s.Close(); close(closed) }()
	select {
	case <-closed:
		t.Fatal("closed before confirmation persisted")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	if err := <-confirmed; err != nil {
		t.Fatal(err)
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not finish")
	}
	if failure := <-repo.finished; failure != context.Canceled.Error() {
		t.Fatal(failure)
	}
	if _, err := s.Send(context.Background(), auth.User{}, "chat", "key", "hello"); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if _, err := s.Retry(context.Background(), auth.User{}, "run"); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if _, err := s.Confirm(context.Background(), auth.User{}, "action", nil); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if _, err := s.Cancel(context.Background(), auth.User{}, "action"); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
}
