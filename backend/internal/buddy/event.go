package buddy

// Event describes live agent activity independently of the HTTP transport.
type Event struct {
	Type     string
	Chat     *Chat
	State    string
	Progress string
	Text     string
	Source   *Source
	Ticket   *Ticket
}
