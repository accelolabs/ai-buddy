package config

import (
	"errors"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTP struct {
	Addr string `env:"HTTP_ADDR" env-default:":8080"`
}
type Database struct {
	URL string `env:"DATABASE_URL" env-required:"true"`
}
type Ollama struct {
	URL string `env:"OLLAMA_URL" env-default:"http://localhost:11434"`
}
type Tracker struct {
	Timezone string `env:"APP_TIMEZONE" env-default:"Europe/Moscow"`
}
type Wiki struct {
	KnowledgeBaseDir string  `env:"KNOWLEDGE_BASE_DIR" env-default:"../knowledge-base"`
	MinSimilarity    float64 `env:"WIKI_MIN_SIMILARITY" env-default:"0.35"`
}
type Config struct {
	HTTP     HTTP
	Database Database
	Ollama   Ollama
	Tracker  Tracker
	Wiki     Wiki
}

func Load() (Config, error) {
	var c Config
	if err := cleanenv.ReadEnv(&c); err != nil {
		return c, errors.New("invalid configuration; check DATABASE_URL and environment value types")
	}
	if strings.TrimSpace(c.Database.URL) == "" || strings.TrimSpace(c.HTTP.Addr) == "" {
		return c, errors.New("DATABASE_URL and HTTP_ADDR must be nonempty")
	}
	_, port, err := net.SplitHostPort(c.HTTP.Addr)
	n, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || n < 0 || n > 65535 {
		return c, errors.New("invalid HTTP_ADDR")
	}

	if _, err := time.LoadLocation(c.Tracker.Timezone); err != nil {
		return c, errors.New("invalid APP_TIMEZONE")
	}
	u, err := url.Parse(c.Ollama.URL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return c, errors.New("invalid OLLAMA_URL")
	}
	v := c.Wiki.MinSimilarity
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
		return c, errors.New("invalid WIKI_MIN_SIMILARITY")
	}
	return c, nil
}
