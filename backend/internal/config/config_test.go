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
