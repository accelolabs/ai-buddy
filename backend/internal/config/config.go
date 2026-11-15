package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Addr             string `env:"ADDR" env-default:":8080"`
	DatabaseURL      string `env:"DATABASE_URL" env-required:"true"`
	KnowledgeBaseDir string `env:"KNOWLEDGE_BASE_DIR" env-default:"../knowledge-base"`
}

func Load() (Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	return cfg, nil
}
