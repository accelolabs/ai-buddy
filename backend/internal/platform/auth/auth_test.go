package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"

	"ai-buddy/internal/platform/config"
)

type brokenStore struct {
	scs.Store
	find, commit, del bool
}

func (s *brokenStore) Find(k string) ([]byte, bool, error) {
	if s.find {
		return nil, false, errors.New("down")
	}

	return s.Store.Find(k)
}

func (s *brokenStore) Commit(k string, b []byte, e time.Time) error {
	if s.commit {
		return errors.New("down")
	}

	return s.Store.Commit(k, b, e)
}

func (s *brokenStore) Delete(k string) error {
	if s.del {
		return errors.New("down")
	}

	return s.Store.Delete(k)
}

type fixture struct {
	a                      *Auth
	provider               *httptest.Server
	store                  *brokenStore
	claims                 map[string]any
	challenge, nonce, mode string
	key                    *rsa.PrivateKey
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	f := &fixture{
		key: key,
		claims: map[string]any{
			"authentik_id":       "42",
			"role":               "intern",
			"team":               "backend",
			"name":               "Alice",
			"email":              "a@example.com",
			"preferred_username": "alice",
		},
	}
	f.provider = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]any{
				"issuer":                                f.provider.URL,
				"authorization_endpoint":                f.provider.URL + "/authorize",
				"token_endpoint":                        f.provider.URL + "/token",
				"jwks_uri":                              f.provider.URL + "/jwks",
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/jwks":
			json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
				Key:       &key.PublicKey,
				KeyID:     "test",
				Algorithm: "RS256",
				Use:       "sig",
			}}})
		case "/token":
			r.ParseForm()
			hash := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if f.mode == "exchange" || base64.RawURLEncoding.EncodeToString(hash[:]) != f.challenge {
				http.Error(w, `{"error":"invalid_grant"}`, 400)
				return
			}
			if f.mode == "missing_token" {
				json.NewEncoder(w).Encode(map[string]any{"access_token": "discard", "token_type": "Bearer"})
				return
			}
			c := map[string]any{
				"iss":   f.provider.URL,
				"sub":   "subject",
				"aud":   "buddy",
				"exp":   time.Now().Add(time.Hour).Unix(),
				"iat":   time.Now().Unix(),
				"nonce": f.nonce,
			}
			for k, v := range f.claims {
				c[k] = v
			}
			switch f.mode {
			case "nonce":
				c["nonce"] = "wrong"
			case "issuer":
				c["iss"] = "https://wrong.example"
			case "audience":
				c["aud"] = "other"
			case "subject":
				c["sub"] = ""
			case "expired":
				c["exp"] = time.Now().Add(-time.Hour).Unix()
			}
			signingKey := key
			if f.mode == "signature" {
				signingKey, _ = rsa.GenerateKey(rand.Reader, 2048)
			}
			alg := jose.RS256
			if f.mode == "algorithm" {
				alg = jose.RS512
			}
			signer, _ := jose.NewSigner(
				jose.SigningKey{Algorithm: alg, Key: signingKey},
				(&jose.SignerOptions{}).WithHeader("kid", "test"),
			)
			payload, _ := json.Marshal(c)
			signed, _ := signer.Sign(payload)
			token, _ := signed.CompactSerialize()
			if f.mode == "malformed" {
				token = "invalid"
			}
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "discard",
				"token_type":   "Bearer",
				"id_token":     token,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.provider.Close)
	s := scs.New()
	ms := memstore.NewWithCleanupInterval(0)
	f.store = &brokenStore{Store: ms}
	s.Store = f.store
	s.Cookie.Name = "__Host-session"
	s.Cookie.Secure = true
	s.Cookie.HttpOnly = true
	s.Cookie.SameSite = http.SameSiteLaxMode
	s.ErrorFunc = func(w http.ResponseWriter, r *http.Request, e error) {
		http.Error(w, "session unavailable", 500)
	}
	ctx := oidc.ClientContext(context.Background(), f.provider.Client())
	provider, err := NewOIDCProvider(
		ctx,
		config.Config{
			Issuer:       f.provider.URL,
			ClientID:     "buddy",
			ClientSecret: "secret",
			RedirectURI:  "https://app.example/api/auth/callback",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	f.a = New(s, provider)
	return f
}

func (f *fixture) request(method, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "https://app.example"+path, nil)
	r = r.WithContext(oidc.ClientContext(r.Context(), f.provider.Client()))
	if cookie != nil {
		r.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	f.a.Handler().ServeHTTP(w, r)
	return w
}

func (f *fixture) login(t *testing.T, cookie *http.Cookie) (*http.Cookie, string) {
	t.Helper()
	w := f.request("GET", "/api/auth/login", cookie)
	if w.Code != 302 {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}

	u, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}

	q := u.Query()
	if q.Get("scope") != "openid profile email ai_buddy" ||
		q.Get("code_challenge_method") != "S256" ||
		q.Get("response_type") != "code" {
		t.Fatal("invalid authorization parameters")
	}

	f.challenge = q.Get("code_challenge")
	f.nonce = q.Get("nonce")
	for _, c := range w.Result().Cookies() {
		if c.Name == "__Host-session" {
			cookie = c
		}
	}

	if cookie == nil {
		t.Fatal("missing cookie")
	}

	return cookie, q.Get("state")
}

func (f *fixture) callback(c *http.Cookie, state string) *httptest.ResponseRecorder {
	return f.request("GET", "/api/auth/callback?code=code&state="+url.QueryEscape(state), c)
}

func TestLoginLifecycle(t *testing.T) {
	f := newFixture(t)
	if w := f.request("GET", "/api/auth/me", nil); w.Code != 401 {
		t.Fatal(w.Code)
	}

	cookie, state := f.login(t, nil)
	w := f.callback(cookie, state)
	if w.Code != 302 || w.Header().Get("Location") != "/" {
		t.Fatalf("callback %d %s", w.Code, w.Body.String())
	}

	var renewed *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "__Host-session" {
			renewed = c
		}
	}

	if renewed == nil ||
		renewed.Value == cookie.Value ||
		!renewed.Secure ||
		!renewed.HttpOnly ||
		renewed.SameSite != http.SameSiteLaxMode ||
		renewed.Path != "/" ||
		renewed.Domain != "" {
		t.Fatal("invalid rotated cookie")
	}

	w = f.request("GET", "/api/auth/me", renewed)
	var u User
	if json.Unmarshal(w.Body.Bytes(), &u) != nil || u.ID != "42" || u.Username != "alice" {
		t.Fatalf("me %d %s", w.Code, w.Body.String())
	}

	f.claims["team"] = "changed"
	if w = f.request("GET", "/api/auth/me", renewed); strings.Contains(w.Body.String(), "changed") {
		t.Fatal("snapshot changed")
	}

	if w = f.callback(cookie, state); w.Code != 400 {
		t.Fatalf("replay %d", w.Code)
	}

	if w = f.request("GET", "/api/auth/me", cookie); w.Code != 401 {
		t.Fatal("old session survived")
	}

	if w = f.request("POST", "/api/auth/logout", renewed); w.Code != 204 {
		t.Fatal(w.Code)
	}

	if w = f.request("GET", "/api/auth/me", renewed); w.Code != 401 {
		t.Fatal("logout failed")
	}

	if w = f.request("POST", "/api/auth/logout", nil); w.Code != 204 {
		t.Fatal("anonymous logout")
	}
}

func TestRejectedCallback(t *testing.T) {
	f := newFixture(t)
	for _, mode := range []string{
		"state",
		"nonce",
		"pkce",
		"expired_attempt",
		"provider_error",
		"missing_code",
		"exchange",
		"missing_token",
		"issuer",
		"audience",
		"expired",
		"signature",
		"algorithm",
		"malformed",
		"subject",
	} {
		t.Run(mode, func(t *testing.T) {
			f.mode = mode
			cookie, state := f.login(t, nil)
			switch mode {
			case "state":
				state = "wrong"
			case "pkce":
				f.challenge = "wrong"
			case "expired_attempt":
				f.updateSession(t, cookie, func(ctx context.Context) {
					f.a.Sessions.Put(ctx, SessionKeyOIDCExpiry, time.Now().Add(-time.Second))
				})
			}

			var w *httptest.ResponseRecorder
			if mode == "provider_error" {
				w = f.request("GET", "/api/auth/callback?error=denied&state="+state, cookie)
			} else if mode == "missing_code" {
				w = f.request("GET", "/api/auth/callback?state="+state, cookie)
			} else {
				w = f.callback(cookie, state)
			}

			if w.Code != 400 {
				t.Fatalf("accepted %s: %d %s", mode, w.Code, w.Body.String())
			}

			if w = f.callback(cookie, state); w.Code != 400 {
				t.Fatal("callback not consumed")
			}

			if w = f.request("GET", "/api/auth/me", cookie); w.Code != 401 {
				t.Fatal("created session")
			}
		})
	}

	if w := f.callback(nil, "state"); w.Code != 400 {
		t.Fatal(w.Code)
	}
}

func TestClaims(t *testing.T) {
	f := newFixture(t)
	for _, tc := range []struct {
		key string
		v   any
	}{
		{"authentik_id", nil},
		{"authentik_id", " "},
		{"authentik_id", 42},
		{"role", nil},
		{"role", "admin"},
		{"team", nil},
		{"team", " "},
		{"team", []string{"backend"}},
	} {
		t.Run(tc.key+"-"+strings.TrimSpace(strings.ReplaceAll(stringMustJSON(tc.v), "\"", "")), func(t *testing.T) {
			old := f.claims[tc.key]
			defer func() {
				f.claims[tc.key] = old
			}()

			f.claims[tc.key] = tc.v
			c, s := f.login(t, nil)
			if w := f.callback(c, s); w.Code != 400 {
				t.Fatalf("accepted claims %d", w.Code)
			}
		})
	}

	f.claims["role"] = "mentor"
	f.claims["team"] = "new-team"
	c, s := f.login(t, nil)
	if w := f.callback(c, s); w.Code != 302 {
		t.Fatal("mentor rejected")
	}
}

func stringMustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestStoreErrors(t *testing.T) {
	f := newFixture(t)
	c, s := f.login(t, nil)
	f.store.find = true
	if w := f.request("GET", "/api/auth/me", c); w.Code != 500 {
		t.Fatal("find error")
	}

	f.store.find = false
	f.store.del = true
	if w := f.callback(c, s); w.Code != 500 {
		t.Fatal("renew error")
	}

	if w := f.request("POST", "/api/auth/logout", c); w.Code != 500 {
		t.Fatal("destroy error")
	}

	f.store.del = false
	f.store.commit = true
	if w := f.request("GET", "/api/auth/login", nil); w.Code != 500 {
		t.Fatal("commit error")
	}
}

func TestSessionConfiguration(t *testing.T) {
	s, stop := NewSessions(nil, config.Config{SessionLifetime: 24 * time.Hour, SessionIdleTimeout: time.Hour})
	defer stop()

	if s.Lifetime != 24*time.Hour ||
		s.IdleTimeout != time.Hour ||
		s.Cookie.Name != "__Host-session" ||
		!s.Cookie.Secure ||
		!s.Cookie.HttpOnly ||
		s.Cookie.SameSite != http.SameSiteLaxMode ||
		s.Cookie.Path != "/" ||
		s.Cookie.Domain != "" {
		t.Fatal("session configuration")
	}

	w := httptest.NewRecorder()
	s.ErrorFunc(w, httptest.NewRequest("GET", "/", nil), errors.New("secret"))
	if w.Code != 500 || strings.Contains(w.Body.String(), "secret") {
		t.Fatal("error handler")
	}
}

func (f *fixture) updateSession(t *testing.T, cookie *http.Cookie, update func(context.Context)) {
	t.Helper()
	ctx, err := f.a.Sessions.Load(context.Background(), cookie.Value)
	if err != nil {
		t.Fatal(err)
	}

	update(ctx)
	if _, _, err = f.a.Sessions.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestAuthenticateMissingParameters(t *testing.T) {
	f := newFixture(t)
	for _, args := range [][3]string{{"", "pkce", "nonce"}, {"code", "", "nonce"}, {"code", "pkce", ""}} {
		if _, err := f.a.oidcProvider.Authenticate(context.Background(), args[0], args[1], args[2]); err == nil {
			t.Fatal("accepted empty parameters")
		}
	}
}
