package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type getFunc func(context.Context, string) (User, error)

func (f getFunc) Get(ctx context.Context, id string) (User, error) {
	return f(ctx, id)
}

func TestMiddleware(t *testing.T) {
	for _, tt := range []struct {
		name, cookie, id string
		err              error
		status           int
	}{
		{name: "missing cookie", status: http.StatusUnauthorized},
		{name: "empty cookie", cookie: "demo_user=", status: http.StatusUnauthorized},
		{name: "malformed encoding", cookie: "demo_user=%ZZ", status: http.StatusBadRequest},
		{name: "selected user", cookie: "demo_user=intern.backend.1", id: "intern.backend.1", status: http.StatusNoContent},
		{name: "encoded ID", cookie: "demo_user=user%20%2B%2F%D1%8F", id: "user +/я", status: http.StatusNoContent},
		{name: "literal plus", cookie: "demo_user=a+b", id: "a+b", status: http.StatusNoContent},
		{name: "unknown user", cookie: "demo_user=missing", id: "missing", err: fmt.Errorf("get user: %w", sql.ErrNoRows), status: http.StatusUnauthorized},
		{name: "database error", cookie: "demo_user=intern", id: "intern", err: errors.New("private database error"), status: http.StatusInternalServerError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != "" {
				req.Header.Set("Cookie", tt.cookie)
			}
			want := User{ID: tt.id, Name: "Стажёр", Role: RoleIntern, Team: "backend"}
			getCalled := false
			repository := getFunc(func(ctx context.Context, id string) (User, error) {
				getCalled = true
				if ctx != req.Context() || id != tt.id {
					t.Fatalf("Get received context %v and ID %q, want request context and ID %q", ctx, id, tt.id)
				}
				return want, tt.err
			})
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				if user, ok := UserFromContext(r.Context()); !ok || user != want {
					t.Fatalf("context user = %+v, %v; want %+v", user, ok, want)
				}
				w.WriteHeader(http.StatusNoContent)
			})
			recorder := httptest.NewRecorder()
			Middleware(repository)(next).ServeHTTP(recorder, req)
			if recorder.Code != tt.status || nextCalled != (tt.status == http.StatusNoContent) || getCalled != (tt.id != "") {
				t.Fatalf("status = %d, nextCalled = %v, getCalled = %v; want status %d", recorder.Code, nextCalled, getCalled, tt.status)
			}
		})
	}
}

func TestUserFromContextWithoutUser(t *testing.T) {
	if user, ok := UserFromContext(context.Background()); ok || user != (User{}) {
		t.Fatalf("UserFromContext = %+v, %v; want zero user and false", user, ok)
	}
}
