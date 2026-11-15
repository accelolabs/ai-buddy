package buddy

import "time"

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type Message struct {
	ID        int64
	ChatID    string
	Role      string
	Content   string
	CreatedAt time.Time
}
