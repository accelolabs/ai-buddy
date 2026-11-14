package seed

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

type finder struct{ err error }

func (f finder) Mentor(_ context.Context, id string) (string, error) {
	if id == "6" {
		return "", f.err
	}
	return "1", f.err
}
func TestSeedTransactions(t *testing.T) {
	srv := api(t, "")
	defer srv.Close()
	for _, fail := range []string{"", "begin", "insert", "commit"} {
		t.Run(fail, func(t *testing.T) {
			db, m, _ := sqlmock.New()
			defer db.Close()
			if fail == "begin" {
				m.ExpectBegin().WillReturnError(errors.New("down"))
			} else {
				m.ExpectBegin()
				for i := 0; i < 3; i++ {
					e := m.ExpectExec("INSERT INTO tickets")
					if fail == "insert" && i == 1 {
						e.WillReturnError(errors.New("down"))
						m.ExpectRollback()
						break
					}
					e.WillReturnResult(sqlmock.NewResult(0, 1))
				}
				if fail != "insert" {
					if fail == "commit" {
						m.ExpectCommit().WillReturnError(errors.New("down"))
					} else {
						m.ExpectCommit()
					}
				}
			}
			err := run(context.Background(), db, Config{URL: srv.URL, Token: "token"}, srv.Client(), finder{}, time.Now())
			if (err != nil) != (fail != "") {
				t.Fatal(err)
			}
			if err := m.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestExternalFailuresBeforeWrite(t *testing.T) {
	for _, mode := range []string{"error", "json", "missing", "inactive", "role", "team", "id", "duplicate-id", "duplicate", "pagination", ""} {
		t.Run(mode, func(t *testing.T) {
			srv := api(t, mode)
			defer srv.Close()
			db, m, _ := sqlmock.New()
			defer db.Close()
			err := run(context.Background(), db, Config{URL: srv.URL, Token: "token"}, srv.Client(), finder{err: errors.New("tracker down")}, time.Now())
			if err == nil {
				t.Fatal("accepted external failure")
			}
			if err := m.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestConfig(t *testing.T) {
	t.Setenv("AUTHENTIK_URL", "http://auth")
	t.Setenv("AUTHENTIK_API_TOKEN", "token")
	t.Setenv("TRACKER_URL", "http://tracker")
	t.Setenv("TICKET_API_KEY", "ticket")
	if _, err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ k, v string }{{"TRACKER_TIMEOUT", "bad"}, {"TRACKER_TIMEOUT", "0s"}, {"AUTHENTIK_API_TOKEN", ""}, {"AUTHENTIK_URL", "http://user:pass@host"}, {"TRACKER_URL", ""}, {"TICKET_API_KEY", ""}} {
		t.Run(tc.k+tc.v, func(t *testing.T) {
			t.Setenv(tc.k, tc.v)
			if _, err := LoadConfig(); err == nil {
				t.Fatal("accepted invalid config")
			}
		})
	}
}

func TestRunWithTracker(t *testing.T) {
	authAPI := api(t, "")
	defer authAPI.Close()
	calls := 0
	trackerAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer ticket" {
			t.Error("wrong Tracker key")
		}
		if calls == 4 {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte(`{"mentor":{"authentik_id":"1"}}`))
	}))
	defer trackerAPI.Close()
	db, m, _ := sqlmock.New()
	defer db.Close()
	c := Config{
		URL:            authAPI.URL,
		Token:          "token",
		TrackerURL:     trackerAPI.URL,
		TicketAPIKey:   "ticket",
		TrackerTimeout: time.Second,
	}
	// Failure on the last lookup must still happen before any transaction begins.
	if err := Run(context.Background(), db, c); err == nil {
		t.Fatal("accepted Tracker failure")
	}
	if calls != 4 {
		t.Fatal(calls)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
