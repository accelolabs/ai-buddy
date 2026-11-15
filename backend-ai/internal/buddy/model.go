// Package buddy runs conversations, confirmed actions and persisted event streams.
package buddy

import (
	"backend-ai/internal/users"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
	ErrInvalid  = errors.New("invalid input")
)

func ID() string {
	var b [16]byte

	_, _ = rand.Read(b[:]) // crypto/rand.Read always fills the buffer or terminates.

	return hex.EncodeToString(b[:])
}

func Title(text string) string {
	r := []rune(strings.Join(strings.Fields(text), " "))
	if len(r) > 80 {
		r = r[:80]
	}

	return string(r)
}

type Chat struct {
	ID        string    `json:"id"`
	Owner     string    `json:"-"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}
type Message struct {
	ID        string    `json:"id"`
	ChatID    string    `json:"-"`
	RunID     string    `json:"run_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
type Run struct {
	ID, ChatID, Owner, MessageID, Status, Error string
	CreatedAt                                   time.Time
}

func (r Run) Active() bool { return r.Status == RunQueued || r.Status == RunRunning }

type Event struct {
	ID   int64
	Kind string
	Data json.RawMessage
}
type Action struct {
	ID     string          `json:"id"`
	RunID  string          `json:"run_id"`
	Owner  string          `json:"-"`
	Kind   string          `json:"kind"`
	Status string          `json:"status"`
	Fields json.RawMessage `json:"fields"`
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}
type Card struct {
	RunID string          `json:"run_id"`
	Kind  string          `json:"kind"`
	Data  json.RawMessage `json:"data"`
}

type History struct {
	Chat     Chat
	Messages []Message
	Runs     []Run
	Actions  []Action
	Cards    []Card
}
type Repository interface {
	CreateChat(context.Context, string) (Chat, error)
	Chats(context.Context, string) ([]Chat, error)
	History(context.Context, string, string) (History, error)
	Start(context.Context, string, string, string, string) (Run, bool, error)
	Retry(context.Context, string, string) (Run, bool, error)
	Run(context.Context, string, string) (Run, error)
	Event(context.Context, string, string, any) error
	Events(context.Context, string, string, int64) ([]Event, Run, error)
	Finish(context.Context, string, string) error
	Draft(context.Context, Action) (Action, error)
	Claim(context.Context, string, string, json.RawMessage) (Action, bool, error)
	Cancel(context.Context, string, string) (Action, error)
	ActionResult(context.Context, string, json.RawMessage, error) (Action, error)
}
type Runner interface {
	Generate(context.Context, users.User, Run) error
}
type Executor interface {
	Execute(context.Context, users.User, Action) (json.RawMessage, error)
}
