package seed

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func api(t *testing.T, mode string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/core/users/" || r.Header.Get("Authorization") != "Bearer token" {
			t.Error("API contract", r.URL, r.Header)
			w.WriteHeader(401)
			return
		}

		if mode == "error" {
			w.WriteHeader(503)
			return
		}

		if mode == "json" {
			w.Write([]byte("bad"))
			return
		}

		u := user{}
		for i, d := range demos {
			if r.URL.Query().Get("username") == d.Username {
				u = user{
					PK:       int64(i + 1),
					Username: d.Username,
					Name:     "Demo",
					Email:    "demo@test",
					Active:   true,
				}
				u.Attributes.Role = d.Role
				u.Attributes.Team = d.Team
			}
		}

		switch mode {
		case "missing":
			u.Username = "somebody.else"
		case "inactive":
			u.Active = false
		case "role":
			u.Attributes.Role = "admin"
		case "team":
			u.Attributes.Team = "wrong"
		case "id":
			u.PK = 0
		case "duplicate-id":
			u.PK = 1
		}

		results := []user{u}
		if mode == "duplicate" {
			results = append(results, u)
		}

		next := 0
		if mode == "pagination" {
			next = 2
		}

		json.NewEncoder(w).Encode(map[string]any{"results": results, "pagination": map[string]int{"next": next}})
	}))
}

func expectSeed(m sqlmock.Sqlmock, fail string) {
	m.ExpectBegin()
	for i := 0; i < 6; i++ {
		e := m.ExpectExec("INSERT INTO profiles")
		if fail == "profile" && i == 2 {
			e.WillReturnError(errors.New("failure"))
			m.ExpectRollback()
			return
		}

		e.WillReturnResult(sqlmock.NewResult(0, 1))
	}

	for i := 0; i < 4; i++ {
		if i < 3 {
			e := m.ExpectExec("INSERT INTO assignments")
			if fail == "assignment" {
				e.WillReturnError(errors.New("failure"))
				m.ExpectRollback()
				return
			}

			e.WillReturnResult(sqlmock.NewResult(0, 1))
		}

		for j := 0; j < 3; j++ {
			e := m.ExpectExec("INSERT INTO tasks")
			if fail == "task" {
				e.WillReturnError(errors.New("failure"))
				m.ExpectRollback()
				return
			}

			e.WillReturnResult(sqlmock.NewResult(0, 1))
		}
	}

	if fail == "commit" {
		m.ExpectCommit().WillReturnError(errors.New("failure"))
	} else {
		m.ExpectCommit()
	}
}

func TestSeed(t *testing.T) {
	srv := api(t, "")
	defer srv.Close()

	for _, fail := range []string{
		"",
		"profile",
		"assignment",
		"task",
		"commit",
		"begin",
	} {
		t.Run(fail, func(t *testing.T) {
			db, m, _ := sqlmock.New()
			defer db.Close()

			if fail == "begin" {
				m.ExpectBegin().WillReturnError(errors.New("failure"))
			} else {
				expectSeed(m, fail)
			}

			e := Run(context.Background(), db, Config{srv.URL, "token"})
			if (e != nil) != (fail != "") {
				t.Fatal(e)
			}

			if e = m.ExpectationsWereMet(); e != nil {
				t.Fatal(e)
			}
		})
	}
}

func TestFetchFailuresDoNotWrite(t *testing.T) {
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
			srv := api(t, mode)
			defer srv.Close()

			db, m, _ := sqlmock.New()
			defer db.Close()

			if e := Run(context.Background(), db, Config{srv.URL, "token"}); e == nil {
				t.Fatal("expected failure")
			}

			if e := m.ExpectationsWereMet(); e != nil {
				t.Fatal(e)
			}
		})
	}

	db, m, _ := sqlmock.New()
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if Run(ctx, db, Config{"http://127.0.0.1:1", "token"}) == nil {
		t.Fatal("cancellation")
	}

	if Run(ctx, db, Config{"://invalid", "token"}) == nil {
		t.Fatal("invalid URL")
	}

	if e := m.ExpectationsWereMet(); e != nil {
		t.Fatal(e)
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
		c, e := LoadConfig()
		if (e == nil) != tc.valid {
			t.Fatal(tc, e)
		}

		if e == nil && strings.HasSuffix(c.URL, "/") {
			t.Fatal(c)
		}
	}
}
