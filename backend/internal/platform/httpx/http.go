package httpx

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"ai-buddy/internal/platform/auth"
)

var ErrInvalid = errors.New("invalid input")

func JSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("write JSON response", "error", err)
	}
}

// Override ServeMux's text errors for unmatched paths and unsupported methods.
func Fallbacks(mux *http.ServeMux, methods map[string][]string) {
	for path, allowed := range methods {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Allow", strings.Join(allowed, ", "))
			JSON(w, 405, map[string]string{"error": "method not allowed"})
		})
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, 404, map[string]string{"error": "not found"})
	})
}

func KeyAuth(next http.Handler, buddyKey, ticketKey string, ticketAllowed bool) http.Handler {
	buddy, ticket := sha256.Sum256([]byte(buddyKey)), sha256.Sum256([]byte(ticketKey))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			JSON(w, 401, map[string]string{"error": "unauthorized"})
			return
		}

		token := sha256.Sum256([]byte(parts[1]))
		isBuddy := subtle.ConstantTimeCompare(token[:], buddy[:]) == 1 && buddyKey != ""
		isTicket := subtle.ConstantTimeCompare(token[:], ticket[:]) == 1 && ticketKey != ""
		if !isBuddy && !isTicket {
			JSON(w, 401, map[string]string{"error": "unauthorized"})
			return
		}

		if !isBuddy && !ticketAllowed {
			JSON(w, 403, map[string]string{"error": "forbidden"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func Owner(r *http.Request) string {
	if strings.HasPrefix(r.URL.Path, "/internal/") {
		return r.PathValue("id")
	}

	u, _ := auth.CurrentUser(r.Context())
	return u.ID
}

func Decode(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil || decoder.Decode(new(any)) != io.EOF {
		return ErrInvalid
	}

	return nil
}
