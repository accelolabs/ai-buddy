package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/platform/config"
)

func TestSharedRouteContracts(t *testing.T) {
	var issuer string
	oidc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/keys"}); err != nil {
			t.Error(err)
		}
	}))
	defer oidc.Close()
	issuer = oidc.URL
	provider, err := auth.NewOIDCProvider(context.Background(), config.Config{Issuer: issuer, ClientID: "test", ClientSecret: "secret", RedirectURI: "https://app.example/api/auth/callback"})
	if err != nil {
		t.Fatal(err)
	}
	db, m, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	sessions := scs.New()
	sessions.Store = memstore.NewWithCleanupInterval(0)
	login := auth.New(sessions, provider)
	handler := Routes(db, login.Handler(), http.NotFoundHandler())
	ctx, err := sessions.Load(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(auth.User{ID: "i", Role: "intern", Team: "backend"})
	sessions.Put(ctx, auth.SessionKeyUser, string(raw))
	token, _, err := sessions.Commit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		code         int
	}{
		{"GET", "/health/live", 204}, {"GET", "/health/ready", 204},
		{"GET", "/api/auth/login", 302}, {"GET", "/api/auth/callback", 400},
		{"GET", "/api/auth/me", 200}, {"POST", "/api/auth/logout", 204},
		{"PUT", "/api/auth/me", 405},
	} {
		if tc.path == "/health/ready" {
			m.ExpectPing()
		}
		r := httptest.NewRequest(tc.method, tc.path, nil)
		r.AddCookie(&http.Cookie{Name: sessions.Cookie.Name, Value: token})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.code {
			t.Fatalf("%s %s: %d %s", tc.method, tc.path, w.Code, w.Body)
		}
	}
	m.ExpectPing().WillReturnError(fmt.Errorf("database offline"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/health/ready", nil))
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
	if err = m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
