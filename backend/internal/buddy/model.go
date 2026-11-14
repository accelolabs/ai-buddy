package buddy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"ai-buddy/internal/platform/auth"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
	ErrInvalid  = errors.New("invalid input")
)

func ID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
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
	ID, Owner, Title string
	CreatedAt        time.Time
}
type Message struct {
	ID, ChatID, RunID, Role, Content string
	CreatedAt                        time.Time
}
type Run struct {
	ID, ChatID, Owner, MessageID, Status, Error string
	CreatedAt                                   time.Time
}

func (r Run) Active() bool { return r.Status == "queued" || r.Status == "running" }

type Event struct {
	ID   int64
	Kind string
	Data json.RawMessage
}
type Action struct {
	ID, RunID, Owner, Kind, Status string
	Fields, Result                 json.RawMessage
	Error                          string
}
type Card struct {
	RunID, Kind string
	Data        json.RawMessage
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
	Recover(context.Context) error
	Draft(context.Context, Action) (Action, error)
	Claim(context.Context, string, string, json.RawMessage, bool) (Action, bool, error)
	ActionResult(context.Context, string, json.RawMessage, error) (Action, error)
}
type Runner interface {
	Generate(context.Context, auth.User, Run) error
}
type Executor interface {
	Execute(context.Context, auth.User, Action) (json.RawMessage, error)
}
