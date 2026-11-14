package authentik

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-buddy/internal/wiki"
)

func TestAuthentik(t *testing.T) {
	mode := "ok"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/core/users/42/" || r.Header.Get("Authorization") != "Bearer readonly" {
			t.Error("identity or credential")
		}
		switch mode {
		case "404":
			w.WriteHeader(404)
		case "503":
			w.WriteHeader(503)
		case "invalid":
			fmt.Fprint(w, "!")
		case "wrong":
			fmt.Fprint(w, `{"pk":43,"is_active":true,"attributes":{"team":"backend"}}`)
		case "inactive":
			fmt.Fprint(w, `{"pk":42,"is_active":false}`)
		default:
			fmt.Fprint(w, `{"pk":42,"is_active":true,"attributes":{"team":"platform"}}`)
		}
	}))
	defer srv.Close()
	a := New(srv.URL, "readonly")
	ctx := context.Background()
	if team, err := a.Team(ctx, "42"); err != nil || team != "platform" {
		t.Fatal(team, err)
	}
	for _, m := range []string{"404", "503", "invalid", "wrong", "inactive"} {
		mode = m
		if _, err := a.Team(ctx, "42"); err == nil {
			t.Fatal(m)
		}
	}
	for _, id := range []string{"../", "0", "-1", "abc"} {
		if _, err := a.Team(ctx, id); !errors.Is(err, wiki.ErrNotFound) {
			t.Fatal(id, err)
		}
	}
	a.URL = ":"
	if _, err := a.Team(ctx, "42"); err == nil {
		t.Fatal("URL")
	}
	a.URL = "http://127.0.0.1:1"
	if _, err := a.Team(ctx, "42"); err == nil {
		t.Fatal("network")
	}
}
