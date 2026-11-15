package indexer

import (
	"reflect"
	"strings"
	"testing"
)

func TestHeadingAnchors(t *testing.T) {
	source := []byte("# Настройка окружения\n\n## API: версии 1.2!\n\n## Настройка окружения\n\n## Настройка окружения-1\n\n## Настройка окружения")
	headings := headingAnchors(source)
	got := make([]string, 0, len(headings))
	for _, heading := range headings {
		got = append(got, heading.anchor)
		if source[heading.start] != '#' {
			t.Fatalf("incorrect heading position: %d", heading.start)
		}
	}
	want := []string{"настройка-окружения", "api-версии-12", "настройка-окружения-1", "настройка-окружения-1-1", "настройка-окружения-2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("anchors = %v, want %v", got, want)
	}
}

func TestChunksShareHeadingAnchor(t *testing.T) {
	content := "Introduction\n\n## Повтор\n" + strings.Repeat("Я", 3500) + "\n\n## Повтор\nДругой раздел"
	chunks := SplitChunks("doc", content)
	if chunks[0].Anchor != "" {
		t.Fatal("introduction must have no anchor")
	}
	for _, chunk := range chunks[1 : len(chunks)-1] {
		if chunk.Anchor != "повтор" {
			t.Fatalf("split section changed anchor: %#v", chunk)
		}
	}
	if chunks[len(chunks)-1].Anchor != "повтор-1" {
		t.Fatalf("duplicate heading anchor: %#v", chunks[len(chunks)-1])
	}
	// Each document has its own slugger state.
	if got := SplitChunks("other", "## Повтор"); got[0].Anchor != "повтор" {
		t.Fatal("slug state leaked between documents")
	}
}
