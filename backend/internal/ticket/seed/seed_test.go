package seed

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"ai-buddy/internal/testutil"
)

type finder struct{ err error }

func (f finder) Mentor(_ context.Context, id string) (string, error) {
	if id == "6" {
		return "", f.err
	}
	return "1", f.err
}
func TestSeedTransactions(t *testing.T) {
	srv := testutil.Authentik(t, "")
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
			srv := testutil.Authentik(t, mode)
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
