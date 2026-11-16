package buddy

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type Message struct {
	Role    string
	Content string
}
