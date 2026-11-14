package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"ai-buddy/internal/users"
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

func Owner(r *http.Request) string {
	u, _ := users.CurrentUser(r.Context())
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
