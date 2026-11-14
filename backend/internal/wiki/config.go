package wiki

import (
	"errors"
	"math"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"

	"ai-buddy/internal/platform/config"
)

type IndexConfig struct {
	config.Database
	Root      string `env:"KNOWLEDGE_BASE_DIR" env-default:"../knowledge-base"`
	OllamaURL string `env:"OLLAMA_URL" env-default:"http://ollama:11434"`
}

type Config struct {
	config.Config
	OllamaURL      string  `env:"OLLAMA_URL" env-default:"http://ollama:11434"`
	BuddyAPIKey    string  `env:"BUDDY_API_KEY" env-required:"true"`
	AuthentikURL   string  `env:"AUTHENTIK_URL" env-required:"true"`
	AuthentikToken string  `env:"AUTHENTIK_READ_TOKEN" env-required:"true"`
	Minimum        float64 `env:"WIKI_MIN_SIMILARITY" env-default:"0.35"`
}

func LoadIndexConfig() (IndexConfig, error) {
	var c IndexConfig
	if cleanenv.ReadEnv(&c) != nil || strings.TrimSpace(c.URL) == "" {
		return c, errors.New("invalid index configuration")
	}

	return c, nil
}

func LoadConfig() (Config, error) {
	var c Config
	if cleanenv.ReadEnv(&c) != nil {
		return c, errors.New("invalid wiki configuration")
	}

	if err := c.Config.Validate(); err != nil {
		return c, err
	}

	if strings.TrimSpace(c.BuddyAPIKey) == "" ||
		strings.TrimSpace(c.AuthentikToken) == "" ||
		strings.TrimSpace(c.AuthentikURL) == "" ||
		math.IsNaN(c.Minimum) || math.IsInf(c.Minimum, 0) || c.Minimum < 0 || c.Minimum > 1 {
		return c, errors.New("invalid wiki dependencies")
	}

	return c, nil
}
