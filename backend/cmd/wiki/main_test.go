package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	baseconfig "ai-buddy/internal/platform/config"
	"ai-buddy/internal/wiki"
)

func TestStartup(t *testing.T) {
	orig := openDatabase
	defer func() { openDatabase = orig }()
	openDatabase = func(context.Context, string) (*sql.DB, error) { return nil, errors.New("down") }
	if serve(context.Background(), wiki.Config{}) == nil {
		t.Fatal("database failure")
	}
	p := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"issuer":"http://` + r.Host + `","authorization_endpoint":"http://` + r.Host + `/authorize","token_endpoint":"http://` + r.Host + `/token","jwks_uri":"http://` + r.Host + `/jwks"}`)); err != nil {
				t.Error(err)
			}
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
		err := serve(ctx, wiki.Config{Config: baseconfig.Config{Issuer: issuer, HTTPAddr: "127.0.0.1:0"}})
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

func TestRunInvalidConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if run() == nil {
		t.Fatal("missing config")
	}
}
