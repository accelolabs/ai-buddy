package indexer

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"

	"ai-buddy/internal/wiki"
)

func digest(value string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
}

func Parse(path, markdown string) wiki.Document {
	source := []byte(markdown)
	tree := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader(source))
	document := wiki.Document{
		ID:       digest(path),
		Path:     path,
		Title:    strings.TrimSuffix(filepath.Base(path), ".md"),
		Markdown: markdown,
		Hash:     digest(markdown),
	}
	document.Sections = []wiki.Section{{Title: document.Title, Anchor: "introduction"}}
	current := &document.Sections[0]
	used := map[string]bool{}
	haveTitle := false

	for node := tree.FirstChild(); node != nil; node = node.NextSibling() {
		if heading, ok := node.(*ast.Heading); ok {
			title := nodeText(heading, source)
			if heading.Level == 1 && !haveTitle {
				document.Title = title
				haveTitle = true
			}

			if current.Text != "" {
				used[current.Anchor] = true
			}
			base := slug(title)
			anchor := base
			for suffix := 1; used[anchor]; suffix++ {
				anchor = fmt.Sprintf("%s-%d", base, suffix)
			}
			used[anchor] = true
			document.Sections = append(document.Sections, wiki.Section{Title: title, Anchor: anchor})
			current = &document.Sections[len(document.Sections)-1]
			continue
		}

		value := nodeText(node, source)
		if value != "" {
			current.Text = strings.TrimSpace(current.Text + "\n\n" + value)
		}
	}

	if document.Sections[0].Text == "" {
		document.Sections = document.Sections[1:]
	}
	for section, content := range document.Sections {
		for _, part := range Split(content.Text) {
			document.Chunks = append(document.Chunks, wiki.Chunk{Section: section, Text: part})
		}
	}

	return document
}

func nodeText(root ast.Node, source []byte) string {
	var content strings.Builder
	if err := ast.Walk(root, func(node ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			if node.Type() == ast.TypeBlock {
				content.WriteByte('\n')
			}
			return ast.WalkContinue, nil
		}

		switch value := node.(type) {
		case *ast.Text:
			content.Write(value.Value(source))
			if value.SoftLineBreak() || value.HardLineBreak() {
				content.WriteByte('\n')
			}
		case *ast.String:
			content.Write(value.Value)
		case *ast.FencedCodeBlock, *ast.CodeBlock:
			content.Write(node.Lines().Value(source))
		}

		return ast.WalkContinue, nil
	}); err != nil {
		slog.Warn("resource cleanup failed", "error", err)
	}

	return strings.TrimSpace(content.String())
}

func slug(title string) string {
	var anchor strings.Builder
	for _, character := range strings.ToLower(title) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || character == '_' || character == '-' {
			anchor.WriteRune(character)
		} else if unicode.IsSpace(character) {
			anchor.WriteByte('-')
		}
	}
	if anchor.Len() == 0 {
		return "section"
	}

	return anchor.String()
}
