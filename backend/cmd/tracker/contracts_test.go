package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"

	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/testutil"
	"ai-buddy/internal/tracker/interns"
	internshttp "ai-buddy/internal/tracker/interns/http"
	internspostgres "ai-buddy/internal/tracker/interns/postgres"
	"ai-buddy/internal/tracker/migrations"
	"ai-buddy/internal/tracker/progress"
	progresshttp "ai-buddy/internal/tracker/progress/http"
	"ai-buddy/internal/tracker/tasks"
	taskshttp "ai-buddy/internal/tracker/tasks/http"
	taskspostgres "ai-buddy/internal/tracker/tasks/postgres"
)

func TestHTTPContracts(t *testing.T) {
	db := testutil.Database(t, migrations.Files)
	if _, err := db.Exec(`INSERT INTO profiles(authentik_id,name,username,email,role,team) VALUES ('i','I','i','i@example.invalid','intern','backend'),('m','M','m','m@example.invalid','mentor','backend'),('other','Other','other','other@example.invalid','intern','platform'); INSERT INTO assignments(intern_id,mentor_id,team) VALUES('i','m','backend')`); err != nil {
		t.Fatal(err)
	}
	ir, tr := internspostgres.New(db), taskspostgres.New(db)
	is := interns.New(ir)
	sessions := scs.New()
	handler := testutil.Contract(t, "tracker", domainRoutes(auth.New(sessions, nil), taskshttp.New(tasks.New(tr, is, time.UTC, time.Now)), internshttp.New(is), progresshttp.New(progress.New(ir, tr, time.UTC, time.Now)), "buddy", "ticket"))
	cookie := func(id, role string) *http.Cookie {
		w := httptest.NewRecorder()
		sessions.LoadAndSave(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			raw, err := json.Marshal(auth.User{ID: id, Role: role, Team: "backend"})
			if err != nil {
				t.Error(err)
			}
			sessions.Put(r.Context(), auth.SessionKeyUser, string(raw))
		})).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		return w.Result().Cookies()[0]
	}
	intern, mentor := cookie("i", "intern"), cookie("m", "mentor")
	call := func(method, path, body, key string, c *http.Cookie, code int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Authorization", key)
		r.Header.Set("Idempotency-Key", "key")
		if c != nil {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != code {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body)
		}
		return w
	}
	body := `{"title":"Read guide","description":"Setup","deadline":"2026-12-01"}`
	created := call("POST", "/api/tasks", body, "", intern, 201)
	var task struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &task); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ method, path, body, role string }{
		{"GET", "/api/tasks", "", "intern"}, {"POST", "/api/tasks", body, "intern"}, {"POST", "/api/tasks/" + task.ID + "/complete", "", "intern"},
		{"GET", "/api/progress", "", "intern"}, {"GET", "/api/interns", "", "mentor"}, {"GET", "/api/interns/i/progress", "", "mentor"},
	} {
		c := intern
		if tc.role == "mentor" {
			c = mentor
		}
		call(tc.method, tc.path, tc.body, "", c, 200)
		call(tc.method, tc.path, tc.body, "", nil, 401)
		wrong := mentor
		if tc.role == "mentor" {
			wrong = intern
		}
		call(tc.method, tc.path, tc.body, "", wrong, 403)
	}
	for _, tc := range []struct {
		method, path, body string
		code               int
	}{
		{"GET", "/internal/interns/i/progress", "", 200}, {"POST", "/internal/interns/i/tasks", body, 200},
		{"POST", "/internal/interns/i/tasks/" + task.ID + "/complete", "", 200}, {"GET", "/internal/interns/i/mentor", "", 200},
	} {
		call(tc.method, tc.path, tc.body, "Bearer buddy", nil, tc.code)
		call(tc.method, tc.path, tc.body, "Bearer wrong", nil, 401)
	}
	call("GET", "/internal/interns/i/mentor", "", "Bearer ticket", nil, 200)
	call("GET", "/internal/interns/i/progress", "", "Bearer ticket", nil, 403)
	call("GET", "/api/interns/other/progress", "", "", mentor, 404)
	call("POST", "/api/tasks", `{"title":"changed","deadline":"2026-12-01"}`, "", intern, 409)
	call("POST", "/api/tasks", `{"title":"Read guide","deadline":"bad"}`, "", intern, 400)
	call("POST", "/api/tasks/absent/complete", "", "", intern, 404)
	cross := httptest.NewRequest("POST", "/api/tasks", strings.NewReader(body))
	cross.AddCookie(intern)
	cross.Header.Set("Origin", "https://other.example.invalid")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cross)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
}
