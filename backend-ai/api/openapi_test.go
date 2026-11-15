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

	paths := object(t, doc["paths"])
	operations := 0
	ids := map[string]bool{}

	for path, item := range paths {
		if strings.HasPrefix(path, "/internal/") || strings.HasPrefix(path, "/api/auth/") {
			t.Fatal("obsolete route", path)
		}

		for method, value := range object(t, item) {
			if method != "get" && method != "post" {
				continue
			}

			operations++
			op := object(t, value)

			id, ok := op["operationId"].(string)
			if !ok || id == "" || ids[id] {
				t.Fatal("missing or repeated operationId", id)
			}

			ids[id] = true

			if len(object(t, op["responses"])) == 0 {
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

	if _, exists := object(t, paths["/api/tasks"])["get"]; exists {
		t.Fatal("obsolete task list")
	}

	for _, path := range []string{"/api/tasks", "/api/tasks/{taskID}/complete", "/api/tickets"} {
		if _, exists := object(t, paths[path])["post"]; !exists {
			t.Fatal("missing intern write", path)
		}
	}

	components := object(t, doc["components"])
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

		for part := range strings.SplitSeq(strings.TrimPrefix(ref, "#/"), "/") {
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
					for scheme := range object(t, requirement) {
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
		for name := range object(t, entries) {
			if !reached["#/components/"+kind+"/"+name] {
				t.Errorf("unused component: %s/%s", kind, name)
			}
		}
	}

	eventPath := object(t, paths["/api/runs/{id}/events"])
	eventOperation := object(t, eventPath["get"])
	events := object(t, eventOperation["x-event-payloads"])

	for _, kind := range []string{"stage", "text", "source", "action", "result", "usage", "done", "error"} {
		if _, ok := events[kind]; !ok {
			t.Error("missing SSE payload", kind)
		}
	}
}

func object(t *testing.T, value any) map[string]any {
	t.Helper()

	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected an object, got %T", value)
	}

	return result
}
