package config

import (
	"strings"
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

	if MustLoad().ClientID != "buddy" {
		t.Fatal("MustLoad")
	}
}

func TestInvalid(t *testing.T) {
	for _, tc := range []struct {
		k, v string
	}{
		{"BUDDY_API_KEY", ""},
		{"TICKET_API_KEY", "buddy-key"},
		{"APP_TIMEZONE", "invalid"},
		{"DATABASE_URL", ""},
		{"DATABASE_URL", "password-secret"},
		{"DATABASE_URL", "mysql://db/test"},
		{"HTTP_ADDR", "oops"},
		{"OIDC_ISSUER", "http://example.com"},
		{"OIDC_ISSUER", "https://secret@example.com"},
		{"OIDC_REDIRECT_URI", "https://example.com/callback#secret"},
		{"OIDC_REDIRECT_URI", "https://example.com/callback?secret=1"},
		{"OIDC_CLIENT_ID", ""},
		{"OIDC_CLIENT_SECRET", ""},
		{"SESSION_LIFETIME", "oops-secret"},
		{"SESSION_LIFETIME", "0s"},
		{"SESSION_IDLE_TIMEOUT", "25h"},
		{"SESSION_IDLE_TIMEOUT", "-1s"},
	} {
		t.Run(tc.k+tc.v, func(t *testing.T) {
			environment(t)
			t.Setenv(tc.k, tc.v)
			_, err := Load()
			if err == nil {
				t.Fatal("accepted invalid config")
			}

			if strings.Contains(err.Error(), "secret") {
				t.Fatal("leaked secret")
			}
		})
	}
}

func TestDatabaseOnly(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://user:secret@db/app")
	t.Setenv("OIDC_ISSUER", "")
	if _, err := LoadDatabase(); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DATABASE_URL", "invalid-secret")
	if _, err := LoadDatabase(); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatal("invalid database error")
	}
}
