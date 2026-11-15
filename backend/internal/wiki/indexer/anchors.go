package indexer

import (
	"bytes"
	"html"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type headingAnchor struct {
	start  int
	anchor string
}

// headingPositionParser also records positions for empty ATX headings, which
// have no source segments in goldmark's AST.
type headingPositionParser struct {
	parser.BlockParser
	positions map[ast.Node]int
}

func (p *headingPositionParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	_, segment := reader.Position()
	node, state := p.BlockParser.Open(parent, reader, pc)
	if node != nil {
		p.positions[node] = segment.Start
	}
	return node, state
}

// headingAnchors uses simple slugs for ordinary Markdown headings.
func headingAnchors(source []byte) []headingAnchor {
	positions := make(map[ast.Node]int)
	blocks := parser.DefaultBlockParsers()
	for i := range blocks {
		block := blocks[i].Value.(parser.BlockParser)
		if slices.Equal(block.Trigger(), []byte{'#'}) {
			blocks[i].Value = &headingPositionParser{BlockParser: block, positions: positions}
		}
	}
	md := goldmark.New(goldmark.WithParser(parser.NewParser(
		parser.WithBlockParsers(blocks...),
		parser.WithInlineParsers(parser.DefaultInlineParsers()...),
		parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...),
	)))
	tree := md.Parser().Parse(text.NewReader(source))
	anchors := make([]headingAnchor, 0)
	occurrences := make(map[string]int)
	// These callbacks write only to memory and cannot fail.
	_ = ast.Walk(tree, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || node.Kind() != ast.KindHeading {
			return ast.WalkContinue, nil
		}
		start := positions[node]
		if node.Lines().Len() > 0 {
			start = node.Lines().At(0).Start
		}
		start = bytes.LastIndexByte(source[:start], '\n') + 1
		var heading strings.Builder
		_ = ast.Walk(node, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			switch child := child.(type) {
			case *ast.Image, *ast.RawHTML:
				return ast.WalkSkipChildren, nil
			case *ast.Text:
				heading.Write(child.Value(source))
				if child.SoftLineBreak() || child.HardLineBreak() {
					heading.WriteByte('\n')
				}
			case *ast.String:
				heading.Write(child.Value)
			case *ast.AutoLink:
				heading.Write(child.Label(source))
			}
			return ast.WalkContinue, nil
		})
		base := strings.Map(func(r rune) rune {
			switch {
			case r == ' ':
				return '-'
			case unicode.IsLetter(r), unicode.IsNumber(r), unicode.IsMark(r), r == '-', r == '_':
				return r
			default:
				return -1
			}
		}, strings.ToLower(html.UnescapeString(heading.String())))
		anchor := base
		for {
			if _, used := occurrences[anchor]; !used {
				break
			}
			occurrences[base]++
			anchor = base + "-" + strconv.Itoa(occurrences[base])
		}
		occurrences[anchor] = 0
		anchors = append(anchors, headingAnchor{start: start, anchor: anchor})
		return ast.WalkSkipChildren, nil
	})
	return anchors
}
