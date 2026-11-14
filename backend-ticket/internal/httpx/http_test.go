package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
