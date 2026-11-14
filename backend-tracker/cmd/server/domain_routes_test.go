package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-buddy/backend-tracker/internal/auth"
	"ai-buddy/backend-tracker/internal/interns"
	internshttp "ai-buddy/backend-tracker/internal/interns/http"
	"ai-buddy/backend-tracker/internal/progress"
	progresshttp "ai-buddy/backend-tracker/internal/progress/http"
	progresspostgres "ai-buddy/backend-tracker/internal/progress/postgres"
	"ai-buddy/backend-tracker/internal/tasks"
	taskshttp "ai-buddy/backend-tracker/internal/tasks/http"
	"github.com/alexedwards/scs/v2"
)

type taskResponse struct {
	ID       string `json:"id"`
	InternID string `json:"intern_id"`
	Title    string `json:"title"`
}

type httpFixture struct {
	handler  http.Handler
	store    *memory
	sessions *scs.SessionManager
}

func newHTTP() httpFixture {
	m := newMemory()
	loc, _ := time.LoadLocation("Europe/Moscow")
	now := func() time.Time {
		return time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	}
	internService := interns.New(m)
	taskService := tasks.New(m, internService, loc, now)
	progressService := progress.New(progresspostgres.New(m, m), loc, now)
	sessions := scs.New()
	a := auth.New(sessions, nil)
	return httpFixture{
		domainRoutes(
			a,
			taskshttp.New(taskService),
			internshttp.New(internService),
			progresshttp.New(progressService),
			"buddy",
			"ticket",
		),
		m,
		sessions,
	}
}

func (f httpFixture) cookie(t *testing.T, id, role string) *http.Cookie {
	t.Helper()
	w := httptest.NewRecorder()
	f.sessions.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := json.Marshal(auth.User{
			ID:   id,
			Role: role,
			Team: "backend",
		})
		f.sessions.Put(r.Context(), auth.SessionKeyUser, string(raw))
	})).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	return w.Result().Cookies()[0]
}

func (f httpFixture) request(method, path, body, key, token string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Idempotency-Key", key)
	if token != "" {
		r.Header.Set("Authorization", token)
	}

	if cookie != nil {
		r.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}

func check(t *testing.T, w *httptest.ResponseRecorder, code int) {
	t.Helper()
	if w.Code != code || !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("want JSON %d: %d %s", code, w.Code, w.Body.String())
	}

	if strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), "SQL") {
		t.Fatal("leaked storage error", w.Body)
	}
}

func TestBrowserAPI(t *testing.T) {
	f := newHTTP()
	intern := f.cookie(t, "i", "intern")
	other := f.cookie(t, "j", "intern")
	mentor := f.cookie(t, "m", "mentor")
	for _, route := range []struct {
		method, path string
	}{
		{"GET", "/api/tasks"},
		{"POST", "/api/tasks"},
		{"POST", "/api/tasks/1/complete"},
		{"GET", "/api/progress"},
		{"GET", "/api/interns"},
		{"GET", "/api/interns/i/progress"},
	} {
		check(t, f.request(route.method, route.path, "", "", "", nil), 401)
	}

	for _, tc := range []struct {
		method, path string
		cookie       *http.Cookie
	}{
		{"GET", "/api/tasks", mentor},
		{"POST", "/api/tasks", mentor},
		{"POST", "/api/tasks/1/complete", mentor},
		{"GET", "/api/progress", mentor},
		{"GET", "/api/interns", intern},
		{"GET", "/api/interns/i/progress", intern},
	} {
		check(t, f.request(tc.method, tc.path, "", "", "", tc.cookie), 403)
	}

	check(t, f.request("GET", "/api/tasks?intern_id=j", "", "", "", intern), 200)
	body := `{"title":"  tasks.Task  ","description":" test ","deadline":"2020-01-01"}`
	w := f.request("POST", "/api/tasks", body, "key", "", intern)
	check(t, w, 201)
	var task taskResponse
	if e := json.Unmarshal(w.Body.Bytes(), &task); e != nil || task.InternID != "i" || task.Title != "tasks.Task" {
		t.Fatal(task, e)
	}

	check(t, f.request("POST", "/api/tasks", body, "key", "", intern), 200)
	check(
		t,
		f.request("POST", "/api/tasks", strings.Replace(body, "tasks.Task", "Other", 1), "key", "", intern),
		409,
	)
	check(t, f.request("POST", "/api/tasks/"+task.ID+"/complete", "", "", "", other), 404)
	w = f.request("POST", "/api/tasks/"+task.ID+"/complete", "", "", "", intern)
	check(t, w, 200)
	check(t, f.request("POST", "/api/tasks/"+task.ID+"/complete", "", "", "", intern), 200)
	check(t, f.request("GET", "/api/progress", "", "", "", intern), 200)
	check(t, f.request("GET", "/api/interns", "", "", "", mentor), 200)
	check(t, f.request("GET", "/api/interns/i/progress", "", "", "", mentor), 200)
	check(t, f.request("GET", "/api/interns/j/progress", "", "", "", mentor), 404)
	check(t, f.request("GET", "/api/tasks", "", "", "", f.cookie(t, "missing", "intern")), 404)
	// Browser callers cannot choose an owner in JSON.
	check(
		t,
		f.request("POST", "/api/tasks", `{"title":"a","deadline":"2026-01-01","intern_id":"j"}`, "x", "", intern),
		400,
	)
}

func TestInternalAPI(t *testing.T) {
	f := newHTTP()
	cookie := f.cookie(t, "i", "intern")
	for _, tc := range []struct {
		method, path string
	}{
		{"GET", "/internal/interns/i/progress"},
		{"POST", "/internal/interns/i/tasks"},
		{"POST", "/internal/interns/i/tasks/1/complete"},
		{"GET", "/internal/interns/i/mentor"},
	} {
		for _, token := range []string{
			"",
			"Bearer wrong",
			"Basic buddy",
			"Bearer",
			"Bearer buddy extra",
		} {
			check(t, f.request(tc.method, tc.path, "", "", token, cookie), 401)
		}

		code := 403
		if strings.HasSuffix(tc.path, "/mentor") {
			code = 200
		}

		check(t, f.request(tc.method, tc.path, "", "", "Bearer ticket", nil), code)
	}

	check(t, f.request("GET", "/internal/interns/i/progress", "", "", "Bearer buddy", nil), 200)
	body := `{"title":"tasks.Task","deadline":"2026-01-01"}`
	w := f.request("POST", "/internal/interns/i/tasks", body, "shared", "Bearer buddy", nil)
	check(t, w, 201)
	var task taskResponse
	json.Unmarshal(w.Body.Bytes(), &task)
	check(t, f.request("POST", "/api/tasks", body, "shared", "", cookie), 200)
	check(t, f.request("POST", "/internal/interns/i/tasks", body, "shared", "Bearer buddy", nil), 200)
	check(
		t,
		f.request(
			"POST",
			"/internal/interns/i/tasks",
			strings.Replace(body, "tasks.Task", "other", 1),
			"shared",
			"Bearer buddy",
			nil,
		),
		409,
	)
	check(
		t,
		f.request("POST", "/internal/interns/j/tasks/"+task.ID+"/complete", "", "", "Bearer buddy", nil),
		404,
	)
	w = f.request("POST", "/internal/interns/i/tasks/"+task.ID+"/complete", "", "", "Bearer buddy", nil)
	check(t, w, 200)
	again := f.request("POST", "/internal/interns/i/tasks/"+task.ID+"/complete", "", "", "Bearer buddy", nil)
	check(t, again, 200)
	if w.Body.String() != again.Body.String() {
		t.Fatal("repeat completion changed")
	}

	w = f.request("GET", "/internal/interns/j/mentor", "", "", "Bearer ticket", nil)
	check(t, w, 200)
	if !strings.Contains(w.Body.String(), `"mentor":null`) {
		t.Fatal(w.Body)
	}

	check(t, f.request("GET", "/internal/interns/missing/mentor", "", "", "Bearer buddy", nil), 404)
}

func TestHTTPValidation(t *testing.T) {
	f := newHTTP()
	cookie := f.cookie(t, "i", "intern")
	for _, body := range []string{
		"",
		`{`,
		`null`,
		`[]`,
		`{} {}`,
		`{"title":"a","deadline":"2026-01-01"} true`,
		`{"title":"a","deadline":"2026-02-30"}`,
		`{"title":" ","deadline":"2026-01-01"}`,
		`{"title":"a"}`,
		`{"title":"a","deadline":"2026-01-01","extra":true}`,
		strings.Repeat("x", (1<<20)+1),
	} {
		check(t, f.request("POST", "/api/tasks", body, "k", "", cookie), 400)
	}

	check(t, f.request("POST", "/api/tasks", `{"title":"a","deadline":"2026-01-01"}`, "", "", cookie), 400)
	check(
		t,
		f.request("POST", "/api/tasks", `{"title":"a","deadline":"2026-01-01"}`, strings.Repeat("k", 257), "", cookie),
		400,
	)
}

func TestHTTPStoreErrors(t *testing.T) {
	for _, tc := range []struct {
		fail, method, path, body string
	}{
		{
			"intern",
			"GET",
			"/internal/interns/i/progress",
			"",
		},
		{
			"mentor",
			"GET",
			"/internal/interns/i/mentor",
			"",
		},
		{
			"tasks",
			"GET",
			"/api/tasks",
			"",
		},
		{
			"create",
			"POST",
			"/internal/interns/i/tasks",
			`{"title":"a","deadline":"2026-01-01"}`,
		},
		{
			"complete",
			"POST",
			"/internal/interns/i/tasks/x/complete",
			"",
		},
		{
			"interns",
			"GET",
			"/api/interns",
			"",
		},
		{
			"assigned",
			"GET",
			"/api/interns/i/progress",
			"",
		},
	} {
		t.Run(tc.fail, func(t *testing.T) {
			f := newHTTP()
			f.store.fail = tc.fail
			role, id := "intern", "i"
			if tc.fail == "interns" || tc.fail == "assigned" {
				role, id = "mentor", "m"
			}

			check(t, f.request(tc.method, tc.path, tc.body, "key", "Bearer buddy", f.cookie(t, id, role)), 500)
		})
	}
}

func TestCrossOrigin(t *testing.T) {
	f := newHTTP()
	r := httptest.NewRequest("POST", "https://backend-tracker.test/api/tasks", nil)
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	r.AddCookie(f.cookie(t, "i", "intern"))
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	check(t, w, 403)
}

func TestJSONRoutingErrors(t *testing.T) {
	f := newHTTP()
	for _, path := range []string{"/api/missing", "/internal/missing"} {
		check(t, f.request("GET", path, "", "", "", nil), 404)
	}

	for _, path := range []string{"/api/tasks", "/api/progress", "/internal/interns/i/mentor"} {
		w := f.request("DELETE", path, "", "", "", nil)
		check(t, w, 405)
		if w.Header().Get("Allow") == "" {
			t.Fatal("missing Allow")
		}
	}
}
