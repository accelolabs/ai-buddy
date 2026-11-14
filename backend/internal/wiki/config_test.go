package wiki

import "testing"

func configEnv(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{"DATABASE_URL": "postgres://localhost/wiki", "OIDC_ISSUER": "https://auth.example/application/o/wiki/", "OIDC_CLIENT_ID": "wiki", "OIDC_CLIENT_SECRET": "secret", "OIDC_REDIRECT_URI": "https://wiki.example/api/auth/callback", "BUDDY_API_KEY": "buddy", "AUTHENTIK_URL": "https://auth.example", "AUTHENTIK_READ_TOKEN": "readonly"} {
		t.Setenv(k, v)
	}
}

func TestConfig(t *testing.T) {
	configEnv(t)
	c, err := LoadConfig()
	if err != nil || c.Minimum != 0.35 {
		t.Fatal(c, err)
	}
	i, err := LoadIndexConfig()
	if err != nil || i.Root != "../knowledge-base" {
		t.Fatal(i, err)
	}
	for _, key := range []string{"BUDDY_API_KEY", "AUTHENTIK_READ_TOKEN", "WIKI_MIN_SIMILARITY", "OIDC_ISSUER"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, ":bad")
			if key == "BUDDY_API_KEY" || key == "AUTHENTIK_READ_TOKEN" {
				t.Setenv(key, "")
			}
			if _, err := LoadConfig(); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	t.Setenv("WIKI_MIN_SIMILARITY", "2")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("minimum")
	}
	for _, key := range []string{"DATABASE_URL"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "")
			if _, err := LoadIndexConfig(); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	t.Setenv("DATABASE_URL", "")
	if _, err := LoadIndexConfig(); err == nil {
		t.Fatal("missing")
	}
}
