package indexer

import (
	"strconv"
	"strings"
	"unicode"

	// Project packages.
	"backend/internal/wiki"
)

const (
	chunkWordCount    = 200
	chunkOverlapWords = 40
)

// GetChunks builds overlapping word windows within each heading's section.
func GetChunks(document wiki.Document, content string) []wiki.Chunk {
	chunks := make([]wiki.Chunk, 0)
	anchor := ""
	occurrences := make(map[string]int)

	var section strings.Builder
	flush := func() {
		text := section.String()
		words := wordSpans(text)

		for start := 0; start < len(words); start += chunkWordCount - chunkOverlapWords {
			end := min(start+chunkWordCount, len(words))
			chunks = append(chunks, wiki.Chunk{
				DocumentID: document.ID,
				Ordinal:    len(chunks),
				Anchor:     anchor,
				Text:       text[words[start].start:words[end-1].end],
			})
			if end == len(words) {
				break
			}
		}

		section.Reset()
	}

	var fence byte
	fenceLength := 0
	content = strings.ReplaceAll(content, "\r\n", "\n")

	for line := range strings.SplitAfterSeq(content, "\n") {
		trimmed := strings.TrimLeft(line, " ")
		if len(line)-len(trimmed) <= 3 && len(trimmed) > 0 && (trimmed[0] == '`' || trimmed[0] == '~') {
			length := len(trimmed) - len(strings.TrimLeft(trimmed, string(trimmed[0])))
			if fence == 0 && length >= 3 {
				fence, fenceLength = trimmed[0], length
			} else if fence == trimmed[0] && length >= fenceLength && strings.TrimSpace(trimmed[length:]) == "" {
				fence = 0
			}
		}

		if fence == 0 {
			if title, ok := headingTitle(line); ok {
				flush()
				anchor = headingSlug(title, occurrences)
			}
		}

		section.WriteString(line)
	}

	flush()

	return chunks
}

type wordSpan struct {
	start int
	end   int
}

// Keep byte offsets so windows preserve Markdown spacing and line breaks.
func wordSpans(text string) []wordSpan {
	words := make([]wordSpan, 0)
	start := -1

	for offset, r := range text {
		if unicode.IsSpace(r) {
			if start >= 0 {
				words = append(words, wordSpan{
					start: start,
					end:   offset,
				})
				start = -1
			}
		} else if start < 0 {
			start = offset
		}
	}

	if start >= 0 {
		words = append(words, wordSpan{
			start: start,
			end:   len(text),
		})
	}

	return words
}

func headingTitle(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 {
		return "", false
	}

	trimmed = strings.TrimRight(trimmed, "\r\n")
	level := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
	if level < 1 || level > 6 || (level < len(trimmed) && trimmed[level] != ' ' && trimmed[level] != '\t') {
		return "", false
	}

	title := strings.TrimSpace(trimmed[level:])
	withoutClosingHashes := strings.TrimRight(title, "#")
	if len(withoutClosingHashes) < len(title) &&
		(withoutClosingHashes == "" ||
			strings.HasSuffix(
				withoutClosingHashes,
				" ",
			) ||
			strings.HasSuffix(
				withoutClosingHashes,
				"\t",
			)) {
		title = strings.TrimSpace(withoutClosingHashes)
	}

	return title, true
}

func headingSlug(title string, occurrences map[string]int) string {
	base := strings.Map(func(r rune) rune {
		switch {
		case r == ' ':
			return '-'
		case unicode.IsLetter(r), unicode.IsNumber(r), unicode.IsMark(r), r == '-', r == '_':
			return r
		default:
			return -1
		}
	}, strings.ToLower(title))
	anchor := base

	for {
		if _, used := occurrences[anchor]; !used {
			break
		}

		occurrences[base]++
		anchor = base + "-" + strconv.Itoa(occurrences[base])
	}

	occurrences[anchor] = 0

	return anchor
}
