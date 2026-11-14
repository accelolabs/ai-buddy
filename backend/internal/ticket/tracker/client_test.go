package tracker

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-buddy/internal/ticket/apperror"
)

func TestMentor(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		code       int
		want       string
		err        error
	}{
		{"assigned", `{"mentor":{"authentik_id":"42"}}`, 200, "42", nil},
		{"unassigned", `{"mentor":null}`, 200, "", nil},
		{"missing", `{}`, 200, "", apperror.ErrTracker},
		{"invalid id", `{"mentor":{"authentik_id":" "}}`, 200, "", apperror.ErrTracker},
		{"wrong type", `{"mentor":{"authentik_id":42}}`, 200, "", apperror.ErrTracker},
		{"malformed", `{"mentor":`, 200, "", apperror.ErrTracker},
		{"trailing", `{"mentor":null}{}`, 200, "", apperror.ErrTracker},
		{"huge", strings.Repeat(" ", 1<<20) + `{}`, 200, "", apperror.ErrTracker},
		{"unknown", `{}`, 404, "", apperror.ErrNotFound},
		{"auth", `{}`, 401, "", apperror.ErrTracker},
		{"down", `{}`, 500, "", apperror.ErrTracker},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.EscapedPath() != "/internal/interns/a%2Fb/mentor" || r.Header.Get("Authorization") != "Bearer secret" || r.Method != "GET" {
					t.Error("wrong request", r.URL)
				}
				w.WriteHeader(tc.code)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()
			got, err := New(server.URL, "secret", time.Second).Mentor(context.Background(), "a/b")
			if got != tc.want || !errors.Is(err, tc.err) {
				t.Fatal(got, err)
			}
		})
	}
}
func TestTimeoutAndRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("redirect followed") }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "slow") {
			<-r.Context().Done()
			return
		}
		http.Redirect(w, r, target.URL, 302)
	}))
	defer server.Close()
	for _, id := range []string{"slow", "redirect"} {
		_, err := New(server.URL, "secret", 20*time.Millisecond).Mentor(context.Background(), id)
		if !errors.Is(err, apperror.ErrTracker) {
			t.Fatal(err)
		}
	}
}
