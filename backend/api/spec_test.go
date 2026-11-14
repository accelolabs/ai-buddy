package api_test

import (
	"ai-buddy/internal/testutil"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestStandaloneSpecifications(t *testing.T) {
	for _, service := range []string{"buddy", "tracker", "ticket", "wiki"} {
		t.Run(service, func(t *testing.T) {
			raw, err := os.ReadFile(service + ".yaml")
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), "nullable:") {
				t.Fatal("use JSON Schema null types in OpenAPI 3.1")
			}
			loader := openapi3.NewLoader()
			loader.IsExternalRefsAllowed = false
			doc, err := loader.LoadFromFile(service + ".yaml")
			if err != nil {
				t.Fatal(err)
			}
			if doc.OpenAPI != "3.1.1" {
				t.Fatal(doc.OpenAPI)
			}
			if err = doc.Validate(context.Background()); err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{"/health/live", "/health/ready", "/api/auth/login", "/api/auth/callback", "/api/auth/me", "/api/auth/logout"} {
				if doc.Paths.Value(path) == nil {
					t.Fatal("missing", path)
				}
			}
		})
	}
}

func TestRequestConstraints(t *testing.T) {
	for _, tc := range []struct {
		service, path, body string
		valid               bool
	}{
		{"buddy", "/api/chats/chat/messages", `{"content":"hello"}`, true},
		{"buddy", "/api/chats/chat/messages", `{"content":"hello","owner":"other"}`, false},
		{"buddy", "/api/chats/chat/messages", `{}`, false},
		{"tracker", "/api/tasks", `{"title":"Read","deadline":"2026-12-01"}`, true},
		{"tracker", "/api/tasks", `{"title":"Read","deadline":"tomorrow"}`, false},
		{"tracker", "/api/tasks", `{"title":"Read","deadline":"2026-12-01","intern_id":"other"}`, false},
		{"ticket", "/api/tickets", `{"question":"Help?","description":"Context"}`, true},
		{"ticket", "/api/tickets", `{"question":"Help?"}`, false},
		{"ticket", "/api/tickets", `{"question":"Help?","description":"Context","recipient_id":"other"}`, false},
		{"wiki", "/api/search", `{"query":"Setup"}`, true},
		{"wiki", "/api/search", `{"query":"Setup","team":"other"}`, false},
	} {
		t.Run(tc.service+tc.body, func(t *testing.T) {
			doc, err := openapi3.NewLoader().LoadFromFile(tc.service + ".yaml")
			if err != nil {
				t.Fatal(err)
			}
			router, err := legacy.NewRouter(doc)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest("POST", tc.path, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Idempotency-Key", "key")
			route, params, err := router.FindRoute(request)
			if err != nil {
				t.Fatal(err)
			}
			err = openapi3filter.ValidateRequest(context.Background(), &openapi3filter.RequestValidationInput{Request: request, Route: route, PathParams: params, Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}})
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}

func TestSSEExamples(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromFile("buddy.yaml")
	if err != nil {
		t.Fatal(err)
	}
	examples := doc.Paths.Value("/api/runs/{id}/events").Get.Responses.Status(200).Value.Content["text/event-stream"].Examples
	for name, example := range examples {
		t.Run(name, func(t *testing.T) {
			handler := testutil.Contract(t, "buddy", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				if _, err := io.WriteString(w, example.Value.Value.(string)); err != nil {
					t.Error(err)
				}
			}))
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/runs/run/events", nil))
		})
	}
}
