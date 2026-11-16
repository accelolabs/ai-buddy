package wiki

type Chunk struct {
	DocumentID string
	Ordinal    int
	Anchor     string
	Text       string
	Embedding  []float64
}
