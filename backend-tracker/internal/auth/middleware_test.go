package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
)

func TestRequireRoles(t *testing.T) {
	for _, tc := range []struct {
		name       string
		role       string
		roles      []string
		raw        string
		storeError bool
		status     int
	}{
		{name: "intern", role: RoleIntern, roles: []string{RoleIntern}, status: 204},
		{name: "mentor", role: RoleMentor, roles: []string{RoleMentor}, status: 204},
		{name: "either intern", role: RoleIntern, roles: []string{RoleIntern, RoleMentor}, status: 204},
		{name: "either mentor", role: RoleMentor, roles: []string{RoleIntern, RoleMentor}, status: 204},
		{name: "wrong role", role: RoleMentor, roles: []string{RoleIntern}, status: 403},
		{name: "empty list", role: RoleIntern, status: 403},
		{name: "unknown allowed role", role: RoleIntern, roles: []string{"admin"}, status: 403},
		{name: "anonymous", roles: []string{RoleIntern}, status: 401},
		{name: "anonymous empty list", status: 401},
		{name: "corrupted snapshot", raw: "invalid", roles: []string{RoleIntern}, status: 500},
		{name: "invalid snapshot", raw: `{"authentik_id":"42","role":"admin","team":"backend"}`, roles: []string{"admin"}, status: 500},
		{name: "store failure", role: RoleIntern, roles: []string{RoleIntern}, storeError: true, status: 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, store := newMiddlewareAuth(t)
			var want User
			raw := tc.raw
			if tc.role != "" {
				want = User{ID: "42", Role: tc.role, Team: "backend", Name: "Alice", Email: "a@example.com", Username: "alice"}
				data, err := json.Marshal(want)
				if err != nil {
					t.Fatal(err)
				}
				raw = string(data)
			}
			var token string
			if raw != "" {
				ctx, err := a.Sessions.Load(context.Background(), "")
				if err != nil {
					t.Fatal(err)
				}
				a.Sessions.Put(ctx, SessionKeyUser, raw)
				token, _, err = a.Sessions.Commit(ctx)
				if err != nil {
					t.Fatal(err)
				}
			}
			store.find = tc.storeError
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				got, ok := CurrentUser(r.Context())
				if !ok || got != want {
					t.Fatalf("user = %+v, want %+v", got, want)
				}
				w.WriteHeader(http.StatusNoContent)
			})
			handler := a.WithSession(a.RequireRoles(next, tc.roles...))
			r := httptest.NewRequest("GET", "https://app.example/api/chat/", nil)
			if token != "" {
				r.AddCookie(&http.Cookie{Name: a.Sessions.Cookie.Name, Value: token})
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tc.status || called != (tc.status == 204) {
				t.Fatalf("status = %d, called = %v; want %d", w.Code, called, tc.status)
			}
		})
	}
}

func newMiddlewareAuth(t *testing.T) (*Auth, *brokenStore) {
	t.Helper()
	memory := memstore.NewWithCleanupInterval(0)
	store := &brokenStore{Store: memory}
	sessions := scs.New()
	sessions.Store = store
	sessions.ErrorFunc = func(w http.ResponseWriter, r *http.Request, err error) { http.Error(w, "session unavailable", 500) }
	return New(sessions, nil), store
}

func TestWithSession(t *testing.T) {
	a, _ := newMiddlewareAuth(t)
	handler := a.WithSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		previous := a.Sessions.GetString(r.Context(), "value")
		a.Sessions.Put(r.Context(), "value", "saved")
		w.Write([]byte(previous))
	}))
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest("GET", "https://app.example/", nil))
	cookies := first.Result().Cookies()
	if first.Code != 200 || len(cookies) != 1 {
		t.Fatalf("first request: %d, cookies %d", first.Code, len(cookies))
	}
	second := httptest.NewRequest("GET", "https://app.example/", nil)
	second.AddCookie(cookies[0])
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, second)
	if w.Code != 200 || w.Body.String() != "saved" {
		t.Fatalf("second request: %d %q", w.Code, w.Body.String())
	}
}

func TestWithSessionCrossOrigin(t *testing.T) {
	a, store := newMiddlewareAuth(t)
	// Rejection happens before reading the session or calling the protected handler.
	store.find = true
	handler := a.WithSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("cross-origin request reached handler") }))
	for _, header := range []struct{ key, value string }{
		{"Sec-Fetch-Site", "cross-site"},
		{"Origin", "https://other.example"},
	} {
		r := httptest.NewRequest("POST", "https://app.example/api/chat/", nil)
		r.Header.Set(header.key, header.value)
		r.AddCookie(&http.Cookie{Name: a.Sessions.Cookie.Name, Value: "session"})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatalf("%s: status %d", header.key, w.Code)
		}
	}
}

func TestWithSessionWriteFailure(t *testing.T) {
	a, store := newMiddlewareAuth(t)
	store.commit = true
	handler := a.WithSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.Sessions.Put(r.Context(), "value", "saved")
		w.WriteHeader(http.StatusNoContent)
	}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "https://app.example/", nil))
	if w.Code != 500 {
		t.Fatalf("status = %d", w.Code)
	}
}
