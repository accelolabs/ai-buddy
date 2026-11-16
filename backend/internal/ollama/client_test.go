package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/embed" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}

		var body struct {
			Model    string `json:"model"`
			Input    string `json:"input"`
			Truncate *bool  `json:"truncate"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}

		if body.Model != "embeddinggemma:300m" || body.Input != "Русский текст" || body.Truncate == nil ||
			*body.Truncate {
			t.Errorf("request body: %+v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"embeddings":[[0.1,0.2,0.3]]}`)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()

	got, err := NewClient(server.URL+"/", "embeddinggemma:300m").Embed(
		context.Background(),
		"Русский текст",
	)
	if err != nil || !reflect.DeepEqual(got, []float64{0.1, 0.2, 0.3}) {
		t.Fatalf("Embed: %v, %v", got, err)
	}
}

func TestEmbedErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"http failure", `{"error":"model not found"}`, 404},
		{"invalid json", `invalid`, 200},
		{"no embeddings", `{}`, 200},
		{"empty vector", `{"embeddings":[[]]}`, 200},
		{"multiple vectors", `{"embeddings":[[1],[2]]}`, 200},
		{"api error", `{"error":"failed","embeddings":[[1]]}`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				if _, err := w.Write([]byte(tc.body)); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()

			if _, err := NewClient(server.URL, "model").Embed(context.Background(), "text"); err == nil {
				t.Fatal("invalid response accepted")
			}
		})
	}
}

func TestEmbedCancellation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
	}))
	defer server.Close()

	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := NewClient(server.URL, "model").Embed(ctx, "text")
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request not started")
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("request not cancelled")
	}
}
