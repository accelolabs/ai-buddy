package buddy

type Recipient struct {
	ID   string
	Name string
}

type Ticket struct {
	Title       string
	Description string
	Recipient   Recipient
}
