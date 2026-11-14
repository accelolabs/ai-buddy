package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-buddy/internal/platform/demousers"
)

var demos = []struct{ Username, Role, Team string }{
	{"mentor.backend", "mentor", "backend"}, {"intern.backend.1", "intern", "backend"}, {"intern.backend.2", "intern", "backend"},
	{"mentor.platform", "mentor", "platform"}, {"intern.platform.1", "intern", "platform"}, {"intern.platform.2", "intern", "platform"},
}

func Authentik(t *testing.T, mode string) *httptest.Server {
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

		u := demousers.User{}
		for i, d := range demos {
			if r.URL.Query().Get("username") == d.Username {
				u = demousers.User{
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

		results := []demousers.User{u}
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
