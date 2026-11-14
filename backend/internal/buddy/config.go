package buddy

import (
	"errors"
	"strings"

	"ai-buddy/internal/buddy/remote"
	"ai-buddy/internal/platform/config"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	config.Config
	BuddyAPIKey   string `env:"BUDDY_API_KEY" env-required:"true"`
	OllamaURL     string `env:"OLLAMA_URL" env-default:"http://ollama:11434"`
	WikiURL       string `env:"WIKI_URL" env-default:"http://backend-wiki:8080"`
	WikiPublicURL string `env:"WIKI_PUBLIC_URL" env-required:"true"`
	TrackerURL    string `env:"TRACKER_URL" env-default:"http://backend-tracker:8080"`
	TicketURL     string `env:"TICKET_URL" env-default:"http://backend-ticket:8080"`
}

func Load() (c Config, err error) {
	if cleanenv.ReadEnv(&c) != nil {
		return c, errors.New("missing or malformed configuration")
	}
	if err = c.Validate(); err != nil {
		return
	}
	if c.ClientID != "buddy" || strings.TrimSpace(c.BuddyAPIKey) == "" {
		return c, errors.New("buddy OIDC client and API key required")
	}
	for _, u := range []string{c.OllamaURL, c.WikiURL, c.WikiPublicURL, c.TrackerURL, c.TicketURL} {
		if !remote.ValidURL(u) {
			return c, errors.New("invalid dependency URL")
		}
	}
	return
}
