package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSystemPrompt(t *testing.T) {
	root := t.TempDir()

	for _, tc := range []struct {
		name, text string
		valid      bool
	}{
		{"russian", "Ты Buddy. Всегда отвечай на русском языке.\n", true},
		{"empty", "", false},
		{"whitespace", " \t\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(root, tc.name)
			if err := os.WriteFile(path, []byte(tc.text), 0o600); err != nil {
				t.Fatal(err)
			}

			got, err := (Buddy{SystemPromptFile: path}).LoadSystemPrompt()
			if tc.valid {
				if err != nil || got != tc.text {
					t.Fatalf("prompt = %q, error = %v", got, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "SYSTEM_PROMPT_FILE") {
				t.Fatalf("expected configuration error, got %v", err)
			}
		})
	}

	for _, path := range []string{"", " \t", filepath.Join(root, "missing"), root} {
		if _, err := (Buddy{SystemPromptFile: path}).LoadSystemPrompt(); err == nil ||
			!strings.Contains(err.Error(), "SYSTEM_PROMPT_FILE") {
			t.Fatalf("path %q: expected configuration error, got %v", path, err)
		}
	}
}
