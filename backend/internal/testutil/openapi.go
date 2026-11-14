package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
)

type specification struct {
	doc    *openapi3.T
	router routers.Router
	err    error
}

var contracts sync.Map

func specificationFor(service string) specification {
	if cached, ok := contracts.Load(service); ok {
		return cached.(specification)
	}
	_, file, _, _ := runtime.Caller(0)
	doc, err := openapi3.NewLoader().LoadFromFile(filepath.Join(filepath.Dir(file), "../../api", service+".yaml"))
	var router routers.Router
	if err == nil {
		err = doc.Validate(context.Background())
	}
	if err == nil {
		router, err = legacy.NewRouter(doc)
	}
	value := specification{doc, router, err}
	actual, _ := contracts.LoadOrStore(service, value)
	return actual.(specification)
}

// Contract wraps existing httptest scenarios, validating the actual request and
// response. Authentication is checked by the real middleware, not by the schema.
func Contract(t *testing.T, service string, next http.Handler) http.Handler {
	t.Helper()
	spec := specificationFor(service)
	if spec.err != nil {
		t.Fatal(spec.err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		if r.Body != nil {
			var err error
			body, err = io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			// The handlers accept headerless JSON; normalize its inferred media type.
			if len(body) > 0 && r.Header.Get("Content-Type") == "" {
				r.Header.Set("Content-Type", "application/json")
			}
		}
		request := r.Clone(r.Context())
		request.Body = io.NopCloser(bytes.NewReader(body))
		capture := &contractWriter{ResponseWriter: w, record: httptest.NewRecorder()}
		next.ServeHTTP(capture, r)
		route, params, err := spec.router.FindRoute(request)
		if err != nil { // Generic 404/405 fallbacks have no OpenAPI operation.
			if capture.record.Code != 404 && capture.record.Code != 405 {
				t.Errorf("undocumented route: %s %s: %v", r.Method, r.URL.Path, err)
			}
			return
		}
		input := &openapi3filter.RequestValidationInput{Request: request, PathParams: params, Route: route, Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}}
		if capture.record.Code < 400 {
			if err := openapi3filter.ValidateRequest(r.Context(), input); err != nil {
				t.Errorf("%s %s request: %v", r.Method, r.URL.Path, err)
			}
		}
		output := &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: capture.record.Code, Header: capture.record.Header(), Options: &openapi3filter.Options{IncludeResponseStatus: true}}
		if strings.HasPrefix(capture.record.Header().Get("Content-Type"), "text/event-stream") {
			validateEvents(t, spec.doc, capture.record.Body.String())
			output.Options.ExcludeResponseBody = true
		} else {
			output.SetBodyBytes(capture.record.Body.Bytes())
		}
		if err := openapi3filter.ValidateResponse(r.Context(), output); err != nil {
			t.Errorf("%s %s response %d: %v", r.Method, r.URL.Path, capture.record.Code, err)
		}
	})
}

type contractWriter struct {
	http.ResponseWriter
	record *httptest.ResponseRecorder
}

func (w *contractWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *contractWriter) WriteHeader(code int) {
	w.record.WriteHeader(code)
	w.ResponseWriter.WriteHeader(code)
}
func (w *contractWriter) Write(b []byte) (int, error) {
	for k := range w.record.Header() {
		delete(w.record.Header(), k)
	}
	for k, v := range w.Header() {
		w.record.Header()[k] = v
	}
	n, err := w.ResponseWriter.Write(b)
	if _, e := w.record.Write(b[:n]); e != nil {
		return n, e
	}
	return n, err
}
func (w *contractWriter) Flush() {
	if err := w.FlushError(); err != nil {
		return
	}
}
func (w *contractWriter) FlushError() error {
	for k := range w.record.Header() {
		delete(w.record.Header(), k)
	}
	for k, v := range w.Header() {
		w.record.Header()[k] = v
	}
	w.record.Flush()
	return http.NewResponseController(w.ResponseWriter).Flush()
}

func validateEvents(t *testing.T, doc *openapi3.T, stream string) {
	t.Helper()
	previous := int64(0)
	for _, frame := range strings.Split(stream, "\n\n") {
		if strings.TrimSpace(frame) == "" || strings.HasPrefix(frame, ":") {
			continue
		}
		var kind, data string
		var id int64
		for _, line := range strings.Split(frame, "\n") {
			k, v, _ := strings.Cut(line, ": ")
			switch k {
			case "id":
				var err error
				id, err = strconv.ParseInt(v, 10, 64)
				if err != nil {
					t.Error(err)
				}
			case "event":
				kind = v
			case "data":
				data = v
			}
		}
		if data == "" {
			continue
		} // A disconnect may leave an incomplete last frame.
		if id <= previous {
			t.Errorf("non-monotone SSE id %d after %d", id, previous)
		}
		previous = id
		if kind == "" {
			t.Error("missing event name")
			continue
		}
		schema := doc.Components.Schemas[strings.ToUpper(kind[:1])+kind[1:]+"Event"]
		if schema == nil {
			t.Errorf("undocumented event %q", kind)
			continue
		}
		var payload any
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			t.Error(err)
			continue
		}
		if err := schema.Value.VisitJSON(payload, openapi3.EnableJSONSchema2020()); err != nil {
			t.Error(fmt.Errorf("%s event: %w", kind, err))
		}
	}
}
