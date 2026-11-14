package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRoutes(t *testing.T) {
	db, m, _ := sqlmock.New(sqlmock.MonitorPingsOption(true))
	defer func() {
		// sqlmock cleanup after assertions; Close is not part of this scenario.
		_ = db.Close()
	}()
	h := Routes(db, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }), http.NotFoundHandler())
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
		checked := h
		checked.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		if w.Code != tc.code {
			t.Fatal(tc.path, w.Code)
		}
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
