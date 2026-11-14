package demousers_test

import (
	"context"
	"strings"
	"testing"

	"ai-buddy/internal/platform/demousers"
	"ai-buddy/internal/testutil"
)

func TestFetchFailures(t *testing.T) {
	for _, mode := range []string{
		"missing",
		"inactive",
		"role",
		"team",
		"id",
		"error",
		"json",
		"duplicate",
		"duplicate-id",
		"pagination",
	} {
		t.Run(mode, func(t *testing.T) {
			srv := testutil.Authentik(t, mode)
			defer srv.Close()

			if _, e := demousers.Load(context.Background(), srv.Client(), demousers.Config{srv.URL, "token"}); e == nil {
				t.Fatal("expected failure")
			}
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := demousers.Load(ctx, demousers.Client(), demousers.Config{"http://127.0.0.1:1", "token"}); err == nil {
		t.Fatal("cancellation")
	}

	if _, err := demousers.Load(ctx, demousers.Client(), demousers.Config{"://invalid", "token"}); err == nil {
		t.Fatal("invalid URL")
	}
}

func TestSeedConfig(t *testing.T) {
	for _, tc := range []struct {
		url, token string
		valid      bool
	}{
		{"https://auth.test/", "token", true},
		{"http://auth.test", "token", true},
		{"", "token", false},
		{"file:///secret", "token", false},
		{"https://auth.test", "", false},
		{"https://user:pass@auth.test", "token", false},
		{"https://auth.test?token=secret", "token", false},
	} {
		t.Setenv("AUTHENTIK_URL", tc.url)
		t.Setenv("AUTHENTIK_API_TOKEN", tc.token)
		c, e := demousers.LoadConfig()
		if (e == nil) != tc.valid {
			t.Fatal(tc, e)
		}

		if e == nil && strings.HasSuffix(c.URL, "/") {
			t.Fatal(c)
		}
	}
}

func TestLoad(t *testing.T) {
	s := testutil.Authentik(t, "")
	defer s.Close()
	users, err := demousers.Load(context.Background(), s.Client(), demousers.Config{URL: s.URL, Token: "token"})
	if err != nil || len(users) != 6 || users[1].Username != "intern.backend.1" {
		t.Fatal(users, err)
	}
}
