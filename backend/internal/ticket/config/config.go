package config

import (
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"

	"ai-buddy/internal/platform/config"
	"ai-buddy/internal/ticket/tracker"
)

type Config struct {
	config.Config
	BuddyAPIKey    string        `env:"BUDDY_API_KEY" env-required:"true"`
	TicketAPIKey   string        `env:"TICKET_API_KEY" env-required:"true"`
	TrackerURL     string        `env:"TRACKER_URL" env-required:"true"`
	TrackerTimeout time.Duration `env:"TRACKER_TIMEOUT" env-default:"5s"`
}

func Load() (Config, error) {
	var c Config
	if cleanenv.ReadEnv(&c) != nil {
		return c, errors.New("missing or malformed configuration")
	}
	if err := c.Validate(); err != nil {
		return c, err
	}
	if strings.TrimSpace(c.BuddyAPIKey) == "" || strings.TrimSpace(c.TicketAPIKey) == "" {
		return c, errors.New("service keys are required")
	}
	if !tracker.ValidURL(c.TrackerURL) || c.TrackerTimeout <= 0 {
		return c, errors.New("invalid tracker configuration")
	}
	return c, nil
}
func MustLoad() Config {
	c, err := Load()
	if err != nil {
		slog.Error("configuration rejected", "error", err)
		os.Exit(1)
	}
	return c
}
