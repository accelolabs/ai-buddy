package indexer

import (
	"backend/internal/wiki"
	"strings"
	"unicode"
)

const maxChunkRunes = 1200

// SplitChunks keeps headings with their sections and prefers paragraph boundaries.
func SplitChunks(documentID, content string) []wiki.Chunk {
	chunks := make([]wiki.Chunk, 0)
	anchor := ""
	var text strings.Builder
	flush := func() {
		runes := []rune(strings.TrimSpace(text.String()))
		for len(runes) > 0 {
			end := min(maxChunkRunes, len(runes))
			if end < len(runes) {
				boundary := 0
				for i := end; i > end/2; i-- {
					if runes[i-1] == '\n' && runes[i-2] == '\n' {
						boundary = i
						break
					}
					if boundary == 0 && unicode.IsSpace(runes[i-1]) {
						boundary = i
					}
				}
				if boundary > 0 {
					end = boundary
				}
			}
			if part := strings.TrimSpace(string(runes[:end])); part != "" {
				chunks = append(chunks, wiki.Chunk{
					DocumentID: documentID, Ordinal: len(chunks), Anchor: anchor, Text: part,
				})
			}
			runes = runes[end:]
		}
		text.Reset()
	}

	source := []byte(strings.ReplaceAll(content, "\r\n", "\n"))
	start := 0
	for _, heading := range headingAnchors(source) {
		text.Write(source[start:heading.start])
		flush()
		anchor = heading.anchor
		start = heading.start
	}
	text.Write(source[start:])
	flush()
	return chunks
}
