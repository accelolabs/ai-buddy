package indexer

import (
	"backend/internal/wiki"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSplitChunks(t *testing.T) {
	content := "Введение\n\n# Статья\nТекст\n\n## Раздел\nПример:\n```md\n# Не заголовок\n```\nКонец\n"
	want := []wiki.Chunk{
		{DocumentID: "doc", Ordinal: 0, Text: "Введение"},
		{DocumentID: "doc", Ordinal: 1, Anchor: "статья", Text: "# Статья\nТекст"},
		{DocumentID: "doc", Ordinal: 2, Anchor: "раздел", Text: "## Раздел\nПример:\n```md\n# Не заголовок\n```\nКонец"},
	}
	if got := SplitChunks("doc", content); !reflect.DeepEqual(got, want) {
		t.Fatalf("SplitChunks = %#v, want %#v", got, want)
	}
	if got := SplitChunks("doc", strings.ReplaceAll(content, "\n", "\r\n")); !reflect.DeepEqual(got, want) {
		t.Fatalf("CRLF: %#v", got)
	}
	if got := SplitChunks("doc", " \n\t "); got == nil || len(got) != 0 {
		t.Fatalf("empty: %#v", got)
	}
}

func TestChunkBoundaries(t *testing.T) {
	for _, content := range []string{
		strings.Repeat("Я", 4000),
		strings.Repeat("Русский абзац со словами.\n\n", 200),
		strings.Repeat("A", 700) + "\n\n" + strings.Repeat("B", 700),
		"~~~markdown\n# Внутри кода\n~~~~\n## Снаружи\nТекст",
	} {
		chunks := SplitChunks("doc", content)
		var combined strings.Builder
		for i, chunk := range chunks {
			if chunk.DocumentID != "doc" || chunk.Ordinal != i || !utf8.ValidString(chunk.Text) || utf8.RuneCountInString(chunk.Text) > maxChunkRunes || strings.TrimSpace(chunk.Text) == "" {
				t.Fatalf("invalid chunk: %#v", chunk)
			}
			combined.WriteString(chunk.Text)
		}
		// Splitting may trim boundary whitespace but must not lose or repeat text.
		normalize := func(s string) string { return strings.Join(strings.Fields(s), "") }
		if normalize(combined.String()) != normalize(content) {
			t.Fatal("text lost or duplicated")
		}
	}
	chunks := SplitChunks("doc", strings.Repeat("A", 700)+"\n\n"+strings.Repeat("B", 700))
	if len(chunks) != 2 || chunks[0].Text != strings.Repeat("A", 700) {
		t.Fatal("paragraph boundary ignored")
	}
}

func TestIndexerPublishesChunks(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "backend"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "backend", "guide.md"), []byte("# Заголовок\nТекст\n\n## Раздел\nДетали"), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	repository := syncFunc(func(_ context.Context, documents []wiki.Document, chunks []wiki.Chunk) error {
		called = true
		if len(documents) != 1 || documents[0].Path != "backend/guide.md" || documents[0].Team != "backend" || len(chunks) != 2 {
			t.Fatalf("documents=%#v chunks=%#v", documents, chunks)
		}
		for _, chunk := range chunks {
			if chunk.DocumentID != documents[0].ID {
				t.Fatal("wrong document relation")
			}
		}
		return nil
	})
	if err := NewIndexer(repository, root).Index(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("repository not called")
	}
}
