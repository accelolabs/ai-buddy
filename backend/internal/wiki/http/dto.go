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

func documentMetadata(d wiki.Document) metadata {
	return metadata{
		ID:    d.ID,
		Path:  d.Path,
		Title: d.Title,
		URL:   "/api/documents/" + d.ID,
	}
}
