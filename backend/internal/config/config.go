package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ChatModel        string `env:"CHAT_MODEL" env-default:"qwen3.5:4b"`
	SystemPromptFile string `env:"SYSTEM_PROMPT_FILE" env-default:"prompts/buddy.md"`
	RAGChunkCount    int    `env:"RAG_CHUNK_COUNT" env-default:"3"`
	Addr             string `env:"ADDR" env-default:":8080"`
	DatabaseURL      string `env:"DATABASE_URL" env-required:"true"`
	KnowledgeBaseDir string `env:"KNOWLEDGE_BASE_DIR" env-default:"../knowledge-base"`
	OllamaURL        string `env:"OLLAMA_URL" env-default:"http://localhost:11434"`
	EmbeddingModel   string `env:"EMBEDDING_MODEL" env-default:"embeddinggemma:300m"`
}

func Load() (Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	if cfg.RAGChunkCount <= 0 {
		return Config{}, fmt.Errorf("RAG_CHUNK_COUNT must be positive")
	}

	return cfg, nil
}
