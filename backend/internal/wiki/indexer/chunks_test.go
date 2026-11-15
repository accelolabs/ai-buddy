package indexer

import (
	"backend/internal/wiki"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGetChunksSections(t *testing.T) {
	content := "Введение\n\n# Статья\nТекст\n\n## Раздел\nПример:\n```md\n# Не заголовок\n```\nКонец\n"
	want := []wiki.Chunk{
		{DocumentID: "doc", Ordinal: 0, Text: "Введение"},
		{DocumentID: "doc", Ordinal: 1, Anchor: "статья", Text: "# Статья\nТекст"},
		{DocumentID: "doc", Ordinal: 2, Anchor: "раздел", Text: "## Раздел\nПример:\n```md\n# Не заголовок\n```\nКонец"},
	}
	for _, text := range []string{content, strings.ReplaceAll(content, "\n", "\r\n")} {
		if got := GetChunks(wiki.Document{ID: "doc"}, text); !reflect.DeepEqual(got, want) {
			t.Fatalf("GetChunks = %#v, want %#v", got, want)
		}
	}
	if got := GetChunks(wiki.Document{ID: "doc"}, " \n\t "); got == nil || len(got) != 0 {
		t.Fatalf("empty: %#v", got)
	}
}

func TestGetChunksWindows(t *testing.T) {
	for _, count := range []int{1, 199, 200, 201, 360, 361, 520} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			words := make([]string, count)
			for i := range words {
				words[i] = fmt.Sprintf("слово%d", i+1)
			}
			content := strings.Join(words, " ")
			chunks := GetChunks(wiki.Document{ID: "doc"}, content)
			wantCount := 1
			if count > 200 {
				wantCount += (count - 200 + 159) / 160
			}
			if len(chunks) != wantCount {
				t.Fatalf("chunks=%d, want %d", len(chunks), wantCount)
			}
			for i, chunk := range chunks {
				start := i * 160
				end := min(start+200, count)
				if chunk.DocumentID != "doc" || chunk.Ordinal != i || chunk.Anchor != "" || !reflect.DeepEqual(strings.Fields(chunk.Text), words[start:end]) {
					t.Fatalf("window %d: %#v", i, chunk)
				}
				if i > 0 {
					previous := strings.Fields(chunks[i-1].Text)
					current := strings.Fields(chunk.Text)
					if !reflect.DeepEqual(previous[len(previous)-40:], current[:40]) {
						t.Fatalf("overlap is not 40 words")
					}
				}
			}
		})
	}
}

func TestGetChunksDoNotCrossHeadings(t *testing.T) {
	content := "Перед заголовком\n\n## Первый\n" + strings.Repeat("текст ", 400) + "\n### Второй\nДругой текст"
	chunks := GetChunks(wiki.Document{ID: "doc"}, content)
	if chunks[0].Anchor != "" || chunks[0].Text != "Перед заголовком" {
		t.Fatalf("introduction: %#v", chunks[0])
	}
	for _, chunk := range chunks[1 : len(chunks)-1] {
		if chunk.Anchor != "первый" || strings.Contains(chunk.Text, "Второй") || strings.Contains(chunk.Text, "Другой") {
			t.Fatalf("window crossed heading: %#v", chunk)
		}
	}
	last := chunks[len(chunks)-1]
	if last.Anchor != "второй" || last.Text != "### Второй\nДругой текст" {
		t.Fatalf("last section: %#v", last)
	}
}

func TestGetChunksAnchors(t *testing.T) {
	content := "# Настройка окружения\n## API: версии 1.2!\n## Настройка окружения\n## Настройка окружения-1\n## Настройка окружения\n## C#\n## Конец ###"
	chunks := GetChunks(wiki.Document{ID: "doc"}, content)
	got := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		got = append(got, chunk.Anchor)
	}
	want := []string{"настройка-окружения", "api-версии-12", "настройка-окружения-1", "настройка-окружения-1-1", "настройка-окружения-2", "c", "конец"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("anchors=%v, want %v", got, want)
	}
	if got := GetChunks(wiki.Document{ID: "other"}, "# Настройка окружения"); got[0].Anchor != "настройка-окружения" {
		t.Fatal("anchors leaked between documents")
	}
}

func TestGetChunksPreservesCodeAndFormatting(t *testing.T) {
	content := "# Пример\n\n~~~markdown\n# Внутри кода\n~~~~\n\n```go\nfunc main() {\n\tfmt.Println(1)\n}\n```\n\n    # Отступ\n\n#СледующийБезПробела\n\n## Следующий\nКонец"
	chunks := GetChunks(wiki.Document{ID: "doc"}, content)
	if len(chunks) != 2 || chunks[0].Anchor != "пример" || chunks[1].Anchor != "следующий" {
		t.Fatalf("chunks: %#v", chunks)
	}
	expected := content[:strings.Index(content, "\n\n## Следующий")]
	if chunks[0].Text != expected {
		t.Fatalf("formatting changed: %q", chunks[0].Text)
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
	ctx := context.Background()
	embedded := make(map[string][]float64)
	embedding := embedFunc(func(gotCtx context.Context, input string) ([]float64, error) {
		if gotCtx != ctx {
			t.Fatal("embedding context changed")
		}
		vector := []float64{float64(len(embedded) + 1)}
		embedded[input] = vector
		return vector, nil
	})
	called := false
	repository := syncFunc(func(_ context.Context, documents []wiki.Document, chunks []wiki.Chunk) error {
		called = true
		if len(documents) != 1 || documents[0].Path != "backend/guide.md" || documents[0].Team != "backend" || len(chunks) != 2 {
			t.Fatalf("documents=%#v chunks=%#v", documents, chunks)
		}
		if len(embedded) != len(chunks) {
			t.Fatal("sync called before all embeddings were generated")
		}
		for _, chunk := range chunks {
			if chunk.DocumentID != documents[0].ID || !reflect.DeepEqual(chunk.Embedding, embedded[chunk.Text]) {
				t.Fatal("wrong document relation")
			}
		}
		return nil
	})
	if err := NewIndexer(repository, embedding, root).Index(ctx); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("repository not called")
	}
}
