package config

import (
	"testing"
	"time"
)

func environment(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"DATABASE_URL":         "postgres://user:secret@localhost/test",
		"BUDDY_API_KEY":        "buddy-key",
		"TICKET_API_KEY":       "ticket-key",
		"APP_TIMEZONE":         "Europe/Moscow",
		"HTTP_ADDR":            ":8080",
		"OIDC_ISSUER":          "https://auth.example/issuer/",
		"OIDC_CLIENT_ID":       "buddy",
		"OIDC_CLIENT_SECRET":   "secret",
		"OIDC_REDIRECT_URI":    "https://buddy.example/api/auth/callback",
		"SESSION_LIFETIME":     "24h",
		"SESSION_IDLE_TIMEOUT": "1h",
	} {
		t.Setenv(k, v)
	}
}

func TestLoad(t *testing.T) {
	environment(t)
	c, err := Load()
	if err != nil ||
		c.HTTPAddr != ":8080" ||
		c.SessionLifetime != 24*time.Hour ||
		c.SessionIdleTimeout != time.Hour {
		t.Fatalf("unexpected config: %v", err)
	}

	if c.ClientID != "buddy" {
		t.Fatal("ClientID")
	}
}

func TestInvalid(t *testing.T) {
	for _, tc := range []struct{ k, v string }{{"BUDDY_API_KEY", " "}, {"TICKET_API_KEY", ""}, {"TICKET_API_KEY", "buddy-key"}, {"APP_TIMEZONE", "invalid"}} {
		t.Run(tc.k+tc.v, func(t *testing.T) {
			environment(t)
			t.Setenv(tc.k, tc.v)
			if _, err := Load(); err == nil {
				t.Fatal("accepted invalid configuration")
			}
		})
	}
	environment(t)
	t.Setenv("DATABASE_URL", "mysql://db/app")
	if _, err := Load(); err == nil {
		t.Fatal("base validation bypassed")
	}
}
