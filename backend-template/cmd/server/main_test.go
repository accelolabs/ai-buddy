package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-buddy/backend-template/internal/config"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestStartup(t *testing.T) {
	orig := openDatabase
	defer func() { openDatabase = orig }()
	openDatabase = func(context.Context, string) (*sql.DB, error) { return nil, errors.New("down") }
	if serve(context.Background(), config.Config{}) == nil {
		t.Fatal("database failure")
	}
	p := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"issuer":"http://` + r.Host + `","authorization_endpoint":"http://` + r.Host + `/authorize","token_endpoint":"http://` + r.Host + `/token","jwks_uri":"http://` + r.Host + `/jwks"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer p.Close()
	for _, issuer := range []string{p.URL + "/invalid", p.URL} {
		db, m, _ := sqlmock.New()
		m.ExpectClose()
		openDatabase = func(context.Context, string) (*sql.DB, error) { return db, nil }
		ctx, cancel := context.WithCancel(context.Background())
		if issuer == p.URL {
			go func() { time.Sleep(50 * time.Millisecond); cancel() }()
		}
		err := serve(ctx, config.Config{Issuer: issuer, HTTPAddr: "127.0.0.1:0"})
		cancel()
		if issuer == p.URL && err != nil {
			t.Fatal(err)
		}
		if issuer != p.URL && err == nil {
			t.Fatal("discovery failure")
		}
		if err = m.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	}
}
func TestRoutes(t *testing.T) {
	db, m, _ := sqlmock.New(sqlmock.MonitorPingsOption(true))
	defer db.Close()
	h := routes(db, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }))
	for _, tc := range []struct {
		path string
		code int
		fail bool
	}{{"/health/live", 204, false}, {"/health/ready", 204, false}, {"/health/ready", 503, true}, {"/api/auth/me", 401, false}} {
		if tc.path == "/health/ready" {
			e := m.ExpectPing()
			if tc.fail {
				e.WillReturnError(errors.New("down"))
			}
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		if w.Code != tc.code {
			t.Fatal(tc.path, w.Code)
		}
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestListenFailure(t *testing.T) {
	if serveHTTP(context.Background(), "invalid address", http.NotFoundHandler()) == nil {
		t.Fatal("listen error")
	}
}
