package buddy

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"ai-buddy/internal/users"
)

type Service struct {
	Repo     Repository
	Runner   Runner
	Executor Executor
	Context  context.Context
	wg       sync.WaitGroup
	mu       sync.Mutex
	closing  bool
}

func (s *Service) Wait() { s.wg.Wait() }

// Close rejects new work before waiting, so Add cannot race a zero-count Wait.
func (s *Service) Close() { s.mu.Lock(); s.closing = true; s.mu.Unlock(); s.wg.Wait() }
func (s *Service) begin() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing {
		return false
	}
	s.wg.Add(1)
	return true
}
func (s *Service) Cancel(ctx context.Context, u users.User, id string) (Action, error) {
	if !s.begin() {
		return Action{}, ErrConflict
	}
	defer s.wg.Done()
	return s.Repo.Cancel(ctx, u.ID, id)
}
func (s *Service) Send(ctx context.Context, u users.User, chat, key, text string) (Run, error) {
	if !s.begin() {
		return Run{}, ErrConflict
	}
	defer s.wg.Done()
	if strings.TrimSpace(key) == "" || len(key) > 200 || strings.TrimSpace(text) == "" || len([]rune(text)) > 2000 {
		return Run{}, ErrInvalid
	}
	r, fresh, err := s.Repo.Start(ctx, u.ID, chat, key, text)
	if err == nil && fresh {
		s.launch(u, r)
	}
	return r, err
}
func (s *Service) Retry(ctx context.Context, u users.User, id string) (Run, error) {
	if !s.begin() {
		return Run{}, ErrConflict
	}
	defer s.wg.Done()
	r, fresh, err := s.Repo.Retry(ctx, u.ID, id)
	if err == nil && fresh {
		s.launch(u, r)
	}
	return r, err
}
func (s *Service) launch(u users.User, r Run) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ctx, cancel := context.WithTimeout(s.Context, 10*time.Minute)
		defer cancel()
		err := s.Runner.Generate(ctx, u, r)
		failure := ""
		if err != nil {
			failure = err.Error()
		}
		save, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if err := s.Repo.Finish(save, r.ID, failure); err != nil {
			slog.Error("persist Buddy run failure", "run_id", r.ID, "error", err)
		}
	}()
}
func (s *Service) Confirm(ctx context.Context, u users.User, id string, fields json.RawMessage) (Action, error) {
	if !s.begin() {
		return Action{}, ErrConflict
	}
	defer s.wg.Done()
	a, execute, err := s.Repo.Claim(ctx, u.ID, id, fields)
	if err != nil || !execute {
		return a, err
	}
	// The immutable action ID is also the downstream idempotency key on every retry.
	op, stop := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer stop()
	result, opErr := s.Executor.Execute(op, u, a)
	save, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	return s.Repo.ActionResult(save, a.ID, result, opErr)
}
