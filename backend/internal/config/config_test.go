package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name                 string
		addr                 string
		databaseURL          string
		wantAddr             string
		wantErr              bool
		knowledgeBaseDir     string
		wantKnowledgeBaseDir string
		ollamaURL            string
		embeddingModel       string
		wantOllamaURL        string
		wantEmbeddingModel   string
	}{
		{
			name:                 "default address",
			databaseURL:          "postgres://localhost/buddy",
			wantAddr:             ":8080",
			wantKnowledgeBaseDir: "../knowledge-base",
			wantOllamaURL:        "http://localhost:11434",
			wantEmbeddingModel:   "embeddinggemma:300m",
		},
		{
			name:                 "explicit address",
			addr:                 "127.0.0.1:9090",
			databaseURL:          "postgres://localhost/buddy",
			wantAddr:             "127.0.0.1:9090",
			knowledgeBaseDir:     "/data/wiki",
			wantKnowledgeBaseDir: "/data/wiki",
			ollamaURL:            "http://ollama:11434",
			embeddingModel:       "custom-model",
			wantOllamaURL:        "http://ollama:11434",
			wantEmbeddingModel:   "custom-model",
		},
		{
			name:    "database URL required",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OLLAMA_URL", tt.ollamaURL)
			t.Setenv("EMBEDDING_MODEL", tt.embeddingModel)
			t.Setenv("ADDR", tt.addr)
			t.Setenv("DATABASE_URL", tt.databaseURL)
			t.Setenv("KNOWLEDGE_BASE_DIR", tt.knowledgeBaseDir)
			for key, value := range map[string]string{
				"ADDR": tt.addr, "DATABASE_URL": tt.databaseURL, "KNOWLEDGE_BASE_DIR": tt.knowledgeBaseDir,
				"OLLAMA_URL": tt.ollamaURL, "EMBEDDING_MODEL": tt.embeddingModel,
			} {
				if value == "" {
					if err := os.Unsetenv(key); err != nil {
						t.Fatal(err)
					}
				}
			}

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && (cfg.Addr != tt.wantAddr || cfg.DatabaseURL != tt.databaseURL) {
				t.Fatalf("Load() = %+v, want Addr %q, DatabaseURL %q", cfg, tt.wantAddr, tt.databaseURL)
			}
			if !tt.wantErr && (cfg.OllamaURL != tt.wantOllamaURL || cfg.EmbeddingModel != tt.wantEmbeddingModel) {
				t.Fatalf("Ollama config = %q/%q, want %q/%q", cfg.OllamaURL, cfg.EmbeddingModel, tt.wantOllamaURL, tt.wantEmbeddingModel)
			}
			if !tt.wantErr && cfg.KnowledgeBaseDir != tt.wantKnowledgeBaseDir {
				t.Fatalf("KnowledgeBaseDir = %q, want %q", cfg.KnowledgeBaseDir, tt.wantKnowledgeBaseDir)
			}
		})
	}
}

func TestRAGChunkCount(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/buddy")
	t.Setenv("RAG_CHUNK_COUNT", "")
	if err := os.Unsetenv("RAG_CHUNK_COUNT"); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil || cfg.RAGChunkCount != 3 {
		t.Fatalf("default = %v, %v", cfg.RAGChunkCount, err)
	}
	for _, value := range []string{"5", "0", "-1", "invalid"} {
		t.Setenv("RAG_CHUNK_COUNT", value)
		cfg, err := Load()
		if value == "5" {
			if err != nil || cfg.RAGChunkCount != 5 {
				t.Fatalf("configured = %v, %v", cfg.RAGChunkCount, err)
			}
		} else if err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
}

func TestChatConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/buddy")
	t.Setenv("CHAT_MODEL", "")
	t.Setenv("SYSTEM_PROMPT_FILE", "")
	for _, key := range []string{"CHAT_MODEL", "SYSTEM_PROMPT_FILE"} {
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := Load()
	if err != nil || cfg.ChatModel != "qwen3.5:4b" || cfg.SystemPromptFile != "prompts/buddy.md" {
		t.Fatalf("defaults: %+v, %v", cfg, err)
	}
	t.Setenv("CHAT_MODEL", "custom")
	t.Setenv("SYSTEM_PROMPT_FILE", "/tmp/prompt.md")
	cfg, err = Load()
	if err != nil || cfg.ChatModel != "custom" || cfg.SystemPromptFile != "/tmp/prompt.md" {
		t.Fatalf("configured: %+v, %v", cfg, err)
	}
}

func TestChatLimits(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/buddy")
	for _, key := range []string{"CHAT_CONTEXT_LENGTH", "CHAT_HISTORY_LIMIT"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := Load()
	if err != nil || cfg.ChatContextLength != 16384 || cfg.ChatHistoryLimit != 6 {
		t.Fatalf("defaults: %+v, %v", cfg, err)
	}
	t.Setenv("CHAT_CONTEXT_LENGTH", "32768")
	t.Setenv("CHAT_HISTORY_LIMIT", "4")
	cfg, err = Load()
	if err != nil || cfg.ChatContextLength != 32768 || cfg.ChatHistoryLimit != 4 {
		t.Fatalf("configured: %+v, %v", cfg, err)
	}
	for _, key := range []string{"CHAT_CONTEXT_LENGTH", "CHAT_HISTORY_LIMIT"} {
		for _, value := range []string{"0", "-1", "invalid"} {
			t.Run(key+"/"+value, func(t *testing.T) {
				t.Setenv(key, value)
				if _, err := Load(); err == nil {
					t.Fatalf("accepted %s=%s", key, value)
				}
			})
		}
	}
}
