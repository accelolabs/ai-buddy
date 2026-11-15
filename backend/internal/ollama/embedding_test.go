package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testEmbedding() []float64 { v := make([]float64, 768); v[0] = 1; return v }
func TestOllama(t *testing.T) {
	ctx := context.Background()
	mode := "ok"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mode == "status" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if mode == "context" {
			w.WriteHeader(http.StatusBadRequest)

			if _, err := fmt.Fprint(
				w,
				`{"error":"input length exceeds context length"}`,
			); err != nil {
				t.Error(err)
			}

			return
		}

		if mode == "invalid" {
			if _, err := fmt.Fprint(w, "!"); err != nil {
				t.Error(err)
			}

			return
		}

		if r.URL.Path == "/api/tags" {
			if mode == "missing" {
				if _, err := fmt.Fprint(w, `{"models":[]}`); err != nil {
					t.Error(err)
				}
			} else {
				if _, err := fmt.Fprint(
					w,
					`{"models":[{"name":"embeddinggemma:300m","digest":"sha256:test"}]}`,
				); err != nil {
					t.Error(err)
				}
			}

			return
		}

		var req struct {
			Model, Input string
			Truncate     *bool
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil || req.Truncate == nil ||
			*req.Truncate ||
			req.Model != EmbeddingModel ||
			req.Input == "" {
			t.Error("bad embed request")
		}

		v := testEmbedding()

		if mode == "dimension" {
			v = v[:4]
		}

		if mode == "zero" {
			v[0] = 0
		}

		if err := json.NewEncoder(w).
			Encode(map[string]any{"embeddings": [][]float64{v}}); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	o := New(srv.URL)

	fp, err := o.Fingerprint(ctx)
	if err != nil || !strings.Contains(fp, "sha256:test") {
		t.Fatal(fp, err)
	}

	if v, err := o.Embed(ctx, "test"); err != nil || len(v) != 768 {
		t.Fatal(v, err)
	}

	for _, m := range []string{"status", "invalid", "missing"} {
		mode = m

		if _, err := o.Fingerprint(ctx); err == nil {
			t.Fatal(m)
		}
	}

	for _, m := range []string{"status", "invalid", "context", "dimension", "zero"} {
		mode = m

		_, err := o.Embed(ctx, "test")
		if err == nil {
			t.Fatal(m)
		}

		if m == "context" && !errors.Is(err, ErrContext) {
			t.Fatal(err)
		}
	}

	o.Base = ":"
	if _, err := o.Fingerprint(ctx); err == nil {
		t.Fatal("URL")
	}

	o.Base = "http://127.0.0.1:1"
	if _, err := o.Embed(ctx, "x"); err == nil {
		t.Fatal("network")
	}
}
