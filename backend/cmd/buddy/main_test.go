package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-buddy/internal/buddy"
	baseconfig "ai-buddy/internal/platform/config"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestStartupAndRecovery(t *testing.T) {
	orig := openDatabase
	defer func() { openDatabase = orig }()
	openDatabase = func(context.Context, string) (*sql.DB, error) { return nil, errors.New("down") }
	if serve(context.Background(), buddy.Config{}) == nil {
		t.Fatal("database failure")
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/.well-known/openid-configuration" {
			if _, err := w.Write([]byte(`{"issuer":"http://` + r.Host + `","authorization_endpoint":"http://` + r.Host + `/authorize","token_endpoint":"http://` + r.Host + `/token","jwks_uri":"http://` + r.Host + `/jwks"}`)); err != nil {
				t.Error(err)
			}
			return
		}
		w.WriteHeader(404)
	}))
	defer provider.Close()
	for _, failure := range []bool{false, true} {
		db, m, _ := sqlmock.New()
		openDatabase = func(context.Context, string) (*sql.DB, error) { return db, nil }
		if failure {
			m.ExpectQuery("SELECT id FROM runs").WillReturnError(errors.New("recovery failed"))
		} else {
			m.ExpectQuery("SELECT id FROM runs").WillReturnRows(sqlmock.NewRows([]string{"id"}))
			m.ExpectExec("UPDATE actions").WillReturnResult(sqlmock.NewResult(0, 0))
		}
		m.ExpectClose()
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		err := serve(ctx, buddy.Config{Config: baseconfig.Config{Issuer: provider.URL, HTTPAddr: "127.0.0.1:0"}})
		cancel()
		if (err != nil) != failure {
			t.Fatal(failure, err)
		}
		if e := m.ExpectationsWereMet(); e != nil {
			t.Fatal(e)
		}
	}
	t.Setenv("DATABASE_URL", "")
	if run() == nil {
		t.Fatal("missing configuration")
	}
}
