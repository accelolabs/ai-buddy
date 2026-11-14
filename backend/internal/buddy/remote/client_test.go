package remote

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBoundedRequests(t *testing.T) {
	for _, v := range []string{"http://host:8080/path", "https://host"} {
		if !ValidURL(v) {
			t.Fatal(v)
		}
	}
	for _, v := range []string{"file:///tmp/x", "http://user:pass@host", "http://host?q=x", "http://host/#x", "http:///"} {
		if ValidURL(v) {
			t.Fatal(v)
		}
	}
	mode := "ok"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" || r.Header.Get("Idempotency-Key") != "once" {
			t.Error(r.Header)
		}
		switch mode {
		case "ok":
			if _, err := fmt.Fprint(w, `{"ok":true}`); err != nil {
				t.Error(err)
			}
		case "large":
			if _, err := fmt.Fprint(w, strings.Repeat("x", (1<<20)+2)); err != nil {
				t.Error(err)
			}
		case "bad":
			if _, err := fmt.Fprint(w, "bad"); err != nil {
				t.Error(err)
			}
		case "redirect":
			http.Redirect(w, r, "/other", http.StatusFound)
		case "error":
			w.WriteHeader(500)
		}
	}))
	defer s.Close()
	c := New(s.URL, "key", 0)
	var out map[string]bool
	ctx := context.Background()
	if e := c.Do(ctx, "POST", "/x", "once", map[string]string{"q": "x"}, &out); e != nil || !out["ok"] {
		t.Fatal(e, out)
	}
	for _, v := range []string{"large", "bad", "redirect", "error"} {
		mode = v
		if e := c.Do(ctx, "GET", "/x", "once", nil, &out); e == nil {
			t.Fatal(v)
		}
	}
	if e := c.Do(ctx, "POST", "/x", "once", make(chan int), &out); e == nil {
		t.Fatal("marshal")
	}
	c.Base = ":"
	if e := c.Do(ctx, "GET", "/", "", nil, &out); e == nil {
		t.Fatal("URL")
	}
	c.Base = s.URL
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if e := c.Do(ctx, "GET", "/", "", nil, &out); e == nil {
		t.Fatal("cancel")
	}
}
