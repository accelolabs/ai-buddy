package indexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func corpus(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "common"), 0755); err != nil {
		t.Fatal(err)
	}
	for path, body := range files {
		p := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestMarkdown(t *testing.T) {
	raw := "# Русский заголовок\n\nВступление.\n\n## Повтор\n\n| Имя | Значение |\n|---|---|\n| таймаут | 5s |\n\n- один\n- два\n\n## Повтор\n\n```go\n" + strings.Repeat("пример()\n", 400) + "```\n\n    indented code\n\n`inline` **важно** [ссылка](https://example.com)\n"
	d := Parse("backend/nested/русский.md", raw)
	if d.Title != "Русский заголовок" || d.Markdown != raw || len(d.Sections) != 3 || d.Sections[2].Anchor != "повтор-1" {
		t.Fatalf("bad parse: %+v", d)
	}
	text := ""
	for _, s := range d.Sections {
		text += s.Text
	}
	for _, want := range []string{"таймаут", "пример()", "indented code", "inline", "важно", "ссылка"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %s", want)
		}
	}
	for _, c := range d.Chunks {
		if len([]rune(c.Text)) > 1200 || c.Section >= 3 {
			t.Fatalf("invalid chunk %+v", c)
		}
	}
	if Parse("platform/nested/русский.md", raw).ID == d.ID {
		t.Fatal("move kept identity")
	}
	if slug("!!!") != "section" || slug("A_B-C 123") != "a_b-c-123" {
		t.Fatal("slug")
	}
	chunks := Split(strings.Repeat("я", 2500))
	if len(chunks) != 3 || len([]rune(chunks[1])) != 1200 {
		t.Fatal("unicode split", len(chunks))
	}
	if len(Split(" \n")) != 0 {
		t.Fatal("empty split")
	}
	if len(Parse("a.md", "").Chunks) != 0 {
		t.Fatal("empty document")
	}
}

func TestReadCorpus(t *testing.T) {
	root := corpus(t, map[string]string{"common/nested/a.md": "# A\n\nтекст", "backend/b.md": "body", "README.md": "ignored", "common/a.txt": "ignored", "common/.secret.md": "ignored", ".private/no.md": "ignored"})
	docs, err := ReadCorpus(root)
	if err != nil || len(docs) != 2 {
		t.Fatal(docs, err)
	}
	if err := os.Symlink("a.txt", filepath.Join(root, "common", "link.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCorpus(root); err == nil {
		t.Fatal("accepted symlink")
	}
	for _, p := range []string{filepath.Join(root, "missing"), filepath.Join(root, "README.md"), t.TempDir()} {
		if _, err := ReadCorpus(p); err == nil {
			t.Fatal("accepted", p)
		}
	}
	link := filepath.Join(t.TempDir(), "root")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCorpus(link); err == nil {
		t.Fatal("accepted symlink root")
	}
}

func TestHeadingOnlySections(t *testing.T) {
	d := Parse("common/test.md", "intro\n\n# First\n\n# Second\n\n## First\n\n## First-1\n\ntext")
	if d.Title != "First" || len(d.Sections) != 5 {
		t.Fatal(d)
	}
	anchors := map[string]bool{}
	for _, s := range d.Sections {
		if anchors[s.Anchor] {
			t.Fatal("duplicate anchor", s.Anchor)
		}
		anchors[s.Anchor] = true
	}
}
