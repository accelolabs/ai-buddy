package wikihttp

import "ai-buddy/internal/wiki"

type metadata struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

type document struct {
	metadata
	Markdown string `json:"markdown"`
}

type result struct {
	DocumentID    string `json:"document_id"`
	DocumentTitle string `json:"document_title"`
	SectionTitle  string `json:"section_title"`
	Anchor        string `json:"anchor"`
	Text          string `json:"text"`
	URL           string `json:"url"`
}

func documentMetadata(d wiki.Document) metadata {
	return metadata{
		ID:    d.ID,
		Path:  d.Path,
		Title: d.Title,
		URL:   "/api/documents/" + d.ID,
	}
}
