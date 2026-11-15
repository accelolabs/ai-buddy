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
	if operations != 22 {
		t.Fatal("operations", operations)
	}
	if _, exists := paths["/api/search"]; exists {
		t.Fatal("obsolete search path")
	}
	if _, exists := paths["/api/tasks"].(map[string]any)["get"]; exists {
		t.Fatal("obsolete task list")
	}
	for _, path := range []string{"/api/tasks", "/api/tasks/{taskID}/complete", "/api/tickets"} {
		if _, exists := paths[path].(map[string]any)["post"]; !exists {
			t.Fatal("missing intern write", path)
		}
	}
	components := doc["components"].(map[string]any)
	reached := map[string]bool{}
	var walk func(any)
	follow := func(ref string) {
		if reached[ref] {
			return
		}
		if !strings.HasPrefix(ref, "#/components/") {
			t.Fatal("unexpected reference", ref)
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
		reached[ref] = true
		walk(target)
	}
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if ref, ok := x["$ref"].(string); ok {
				follow(ref)
			}
			if security, ok := x["security"].([]any); ok {
				for _, requirement := range security {
					for scheme := range requirement.(map[string]any) {
						follow("#/components/securitySchemes/" + scheme)
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
	// Traverse references from public roots, including x-event-payloads; scanning
	// components themselves would incorrectly count unreachable definitions as used.
	roots := map[string]any{}
	for key, value := range doc {
		if key != "components" {
			roots[key] = value
		}
	}
	walk(roots)
	for kind, entries := range components {
		for name := range entries.(map[string]any) {
			if !reached["#/components/"+kind+"/"+name] {
				t.Errorf("unused component: %s/%s", kind, name)
			}
		}
	}
	events := paths["/api/runs/{id}/events"].(map[string]any)["get"].(map[string]any)["x-event-payloads"].(map[string]any)
	for _, kind := range []string{"stage", "text", "source", "action", "result", "usage", "done", "error"} {
		if _, ok := events[kind]; !ok {
			t.Error("missing SSE payload", kind)
		}
	}
}
