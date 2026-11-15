package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	for _, k := range []string{"DATABASE_URL", "HTTP_ADDR", "OLLAMA_URL", "SYSTEM_PROMPT_FILE", "APP_TIMEZONE", "KNOWLEDGE_BASE_DIR", "WIKI_MIN_SIMILARITY"} {
		t.Setenv(k, "")

		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("DATABASE_URL", "postgres://localhost/buddy")
	t.Setenv("SYSTEM_PROMPT_FILE", "/custom/prompt.txt")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if c.HTTP.Addr != ":8080" || c.Ollama.URL != "http://localhost:11434" ||
		c.Buddy.SystemPromptFile != "/custom/prompt.txt" ||
		c.Tracker.Timezone != "Europe/Moscow" ||
		c.Wiki.KnowledgeBaseDir != "../knowledge-base" ||
		c.Wiki.MinSimilarity != 0.35 {
		t.Fatal(c)
	}

	t.Setenv("SYSTEM_PROMPT_FILE", "")

	if _, err := Load(); err != nil {
		t.Fatalf("configuration for administrative commands requires no prompt: %v", err)
	}

	for _, tc := range []struct{ k, v string }{{"HTTP_ADDR", "bad"}, {"DATABASE_URL", " "}, {"APP_TIMEZONE", "unknown"}, {"OLLAMA_URL", "file:///tmp"}, {"WIKI_MIN_SIMILARITY", "NaN"}, {"WIKI_MIN_SIMILARITY", "1.1"}} {
		t.Run(tc.k+tc.v, func(t *testing.T) {
			t.Setenv(tc.k, tc.v)

			if _, err := Load(); err == nil {
				t.Fatal("accepted invalid configuration")
			}
		})
	}
}
