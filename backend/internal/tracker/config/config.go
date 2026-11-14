package config

import (
	"errors"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"

	"ai-buddy/internal/platform/config"
)

type Config struct {
	config.Config
	BuddyAPIKey  string `env:"BUDDY_API_KEY" env-required:"true"`
	TicketAPIKey string `env:"TICKET_API_KEY" env-required:"true"`
	AppTimezone  string `env:"APP_TIMEZONE" env-default:"Europe/Moscow"`
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
	if c.BuddyAPIKey == c.TicketAPIKey {
		return c, errors.New("distinct service keys are required")
	}
	if _, err := time.LoadLocation(c.AppTimezone); err != nil {
		return c, errors.New("invalid APP_TIMEZONE")
	}
	return c, nil
}
