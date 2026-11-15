package buddy

type Source struct {
	DocumentID string
	Ordinal    int
	Title      string
	Anchor     string
	Quote      string
}

type Answer struct {
	Content string
	Source  *Source
}
