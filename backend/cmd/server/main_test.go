package main

import (
	"ai-buddy/internal/testutil"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestInvalidArguments(t *testing.T) {
	for _, args := range [][]string{{"unknown"}, {"serve", "extra"}, {"seed", "extra"}, {"migrate"}, {"migrate", "invalid"}, {"index", "--unknown"}, {"index", "extra"}} {
		if err := run(context.Background(), args); err == nil {
			t.Fatal("accepted", args)
		}
	}
}

func TestCommands(t *testing.T) {
	db := testutil.Database(t)
	testutil.Indexing(t, db)

	var schema string
	if err := db.QueryRowContext(context.Background(), `SELECT current_schema()`).
		Scan(&schema); err != nil {
		t.Fatal(err)
	}

	dsn, err := url.Parse(os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}

	query := dsn.Query()
	query.Set("search_path", schema+",public")
	dsn.RawQuery = query.Encode()
	t.Setenv("DATABASE_URL", dsn.String())
	t.Setenv("HTTP_ADDR", "127.0.0.1:0")

	root := t.TempDir()
	if err = os.Mkdir(filepath.Join(root, "common"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err = os.WriteFile(
		filepath.Join(root, "common", "guide.md"),
		[]byte("# Guide\n\nRead this guide."),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	t.Setenv("KNOWLEDGE_BASE_DIR", root)

	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var response any

		switch r.URL.Path {
		case "/api/tags":
			response = map[string]any{
				"models": []map[string]string{
					{"name": "embeddinggemma:300m", "digest": "test"},
				},
			}
		case "/api/embed":
			v := make([]float64, 768)
			v[0] = 1
			response = map[string]any{"embeddings": [][]float64{v}}
		default:
			t.Error("unexpected model request", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)

			return
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	defer model.Close()

	t.Setenv("OLLAMA_URL", model.URL)

	for _, args := range [][]string{{"migrate", "status"}, {"migrate", "down"}, {"migrate", "up"}, {"seed"}, {"seed"}, {"index"}, {"index", "--full"}} {
		err = run(context.Background(), args)
		if err != nil {
			t.Fatal(args, err)
		}
	}

	var count int
	if err = db.QueryRowContext(context.Background(), `SELECT count(*) FROM profiles`).
		Scan(&count); err != nil ||
		count != 6 {
		t.Fatal(count, err)
	}

	if err = db.QueryRowContext(context.Background(), `SELECT count(*) FROM documents`).
		Scan(&count); err != nil ||
		count != 1 {
		t.Fatal(count, err)
	}
}
