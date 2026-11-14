// Package wiki defines the source projection emitted by Buddy tools.
package wiki

type Source struct {
	DocumentID string `json:"document_id"`
	Title      string `json:"document_title"`
	Section    string `json:"section_title"`
	Text       string `json:"text"`
	URL        string `json:"url"`
}
