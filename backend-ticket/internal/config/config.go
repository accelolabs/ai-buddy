package config

import (
	"errors"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"ai-buddy/backend-ticket/internal/tracker"
	"github.com/ilyakaznacheev/cleanenv"
)

type Database struct {
	URL string `env:"DATABASE_URL" env-required:"true"`
}
type Config struct {
	Database
	BuddyAPIKey        string        `env:"BUDDY_API_KEY" env-required:"true"`
	TicketAPIKey       string        `env:"TICKET_API_KEY" env-required:"true"`
	TrackerURL         string        `env:"TRACKER_URL" env-required:"true"`
	TrackerTimeout     time.Duration `env:"TRACKER_TIMEOUT" env-default:"5s"`
	HTTPAddr           string        `env:"HTTP_ADDR" env-default:":8080"`
	Issuer             string        `env:"OIDC_ISSUER" env-required:"true"`
	ClientID           string        `env:"OIDC_CLIENT_ID" env-required:"true"`
	ClientSecret       string        `env:"OIDC_CLIENT_SECRET" env-required:"true"`
	RedirectURI        string        `env:"OIDC_REDIRECT_URI" env-required:"true"`
	SessionLifetime    time.Duration `env:"SESSION_LIFETIME" env-default:"24h"`
	SessionIdleTimeout time.Duration `env:"SESSION_IDLE_TIMEOUT" env-default:"1h"`
}

func LoadDatabase() (Database, error) {
	var c Database
	if cleanenv.ReadEnv(&c) != nil || !validDatabase(c.URL) {
		return c, errors.New("invalid DATABASE_URL")
	}
	return c, nil
}
func validDatabase(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") && u.Hostname() != ""
}
func Load() (Config, error) {
	var c Config
	if cleanenv.ReadEnv(&c) != nil {
		return c, errors.New("missing or malformed configuration")
	}
	_, _, addrErr := net.SplitHostPort(c.HTTPAddr)
	if !tracker.ValidURL(c.TrackerURL) || c.TrackerTimeout <= 0 || strings.TrimSpace(c.BuddyAPIKey) == "" || strings.TrimSpace(c.TicketAPIKey) == "" || addrErr != nil || !validDatabase(c.URL) ||
		strings.TrimSpace(c.ClientID) == "" || strings.TrimSpace(c.ClientSecret) == "" ||
		!validHTTPS(c.Issuer) || !validHTTPS(c.RedirectURI) ||
		c.SessionLifetime <= 0 || c.SessionIdleTimeout <= 0 || c.SessionIdleTimeout > c.SessionLifetime {
		return c, errors.New("invalid HTTP, database, OIDC or session configuration")
	}
	return c, nil
}
func MustLoad() Config {
	c, e := Load()
	if e != nil {
		slog.Error("configuration rejected", "error", e)
		os.Exit(1)
	}
	return c
}

func validHTTPS(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Hostname() != "" &&
		u.User == nil && u.Fragment == "" && u.RawQuery == ""
}
