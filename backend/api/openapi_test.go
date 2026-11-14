package api

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPI(t *testing.T) {
	data, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err = yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["openapi"] != "3.1.1" {
		t.Fatal("expected OpenAPI 3.1.1")
	}
	paths := doc["paths"].(map[string]any)
	operations := 0
	ids := map[string]bool{}
	for path, item := range paths {
		if strings.HasPrefix(path, "/internal/") || strings.HasPrefix(path, "/api/auth/") {
			t.Fatal("obsolete route", path)
		}
		for method, value := range item.(map[string]any) {
			if method != "get" && method != "post" {
				continue
			}
			operations++
			op := value.(map[string]any)
			id, _ := op["operationId"].(string)
			if id == "" || ids[id] {
				t.Fatal("missing or repeated operationId", id)
			}
			ids[id] = true
			if len(op["responses"].(map[string]any)) == 0 {
				t.Fatal("missing responses", path)
			}
		}
	}
	if operations != 24 {
		t.Fatal("operations", operations)
	}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if ref, ok := x["$ref"].(string); ok {
				if !strings.HasPrefix(ref, "#/") {
					t.Fatal("unexpected external reference", ref)
				}
				var target any = doc
				for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
					m, ok := target.(map[string]any)
					if !ok {
						t.Fatal("invalid reference", ref)
					}
					target, ok = m[part]
					if !ok {
						t.Fatal("missing reference", ref)
					}
				}
			}
			for _, value := range x {
				walk(value)
			}
		case []any:
			for _, value := range x {
				walk(value)
			}
		}
	}
	walk(doc)
}
