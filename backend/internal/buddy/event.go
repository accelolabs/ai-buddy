package buddy

// Event describes live agent activity independently of the HTTP transport.
type Event struct {
	Type     string
	State    string
	Progress string
	Text     string
	Source   *Source
}
