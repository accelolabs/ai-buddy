package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-buddy/backend-ticket/internal/apperror"
)

func TestFailure(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code int
	}{
		{apperror.ErrInvalid, 400},
		{apperror.ErrNotFound, 404},
		{apperror.ErrConflict, 409},
		{errors.New("SQL secret"), 500},
	} {
		w := httptest.NewRecorder()
		Failure(w, fmt.Errorf("wrapped: %w", tc.err))
		if w.Code != tc.code ||
			w.Header().Get("Content-Type") != "application/json" ||
			strings.Contains(w.Body.String(), "secret") {
			t.Fatal(w)
		}
	}
}

func TestDecode(t *testing.T) {
	type input struct {
		Title string `json:"title"`
	}
	for _, tc := range []struct {
		body  string
		valid bool
	}{
		{`{"title":"Task"}`, true},
		{`{"title":"Task"} `, true},
		{`{"extra":1}`, false},
		{`{`, false},
		{`{} {}`, false},
		{`[]`, false},
		{strings.Repeat("x", (1<<20)+1), false},
	} {
		var in input
		w := httptest.NewRecorder()
		err := Decode(w, httptest.NewRequest("POST", "/", strings.NewReader(tc.body)), &in)
		if (err == nil) != tc.valid {
			t.Fatal(tc, err)
		}
	}
}

func TestKeyAuth(t *testing.T) {
	for _, ticketAllowed := range []bool{false, true} {
		called := false
		h := KeyAuth(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(204)
			}),
			"buddy",
			"ticket",
			ticketAllowed,
		)
		for _, token := range []string{
			"",
			"Basic buddy",
			"Bearer wrong",
			"Bearer buddy",
			"Bearer ticket",
			"Bearer buddy extra",
		} {
			called = false
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Authorization", token)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			want := 401
			if token == "Bearer buddy" || (token == "Bearer ticket" && ticketAllowed) {
				want = 204
			} else if token == "Bearer ticket" {
				want = 403
			}

			if w.Code != want || called != (want == 204) {
				t.Fatal(token, ticketAllowed, w.Code, called)
			}
		}
	}
}

func TestFallbacksAndOwner(t *testing.T) {
	mux := http.NewServeMux()
	Fallbacks(mux, map[string][]string{"/known": {"GET", "POST"}})
	for _, path := range []string{"/known", "/missing"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("DELETE", path, nil))
		want := 404
		if path == "/known" {
			want = 405
			if w.Header().Get("Allow") != "GET, POST" {
				t.Fatal(w)
			}
		}

		if w.Code != want {
			t.Fatal(w)
		}
	}

	r := httptest.NewRequest("GET", "/internal/interns/i/mentor", nil)
	r.SetPathValue("id", "i")
	if Owner(r) != "i" {
		t.Fatal("path owner")
	}

	r = httptest.NewRequest("GET", "/api/tasks?intern_id=spoof", nil)
	if Owner(r) != "" {
		t.Fatal("untrusted owner")
	}
}
