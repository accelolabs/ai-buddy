package trackerhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/testutil"
	"ai-buddy/internal/tracker/migrations"
	"ai-buddy/internal/tracker/postgres"
	"ai-buddy/internal/tracker/progress"
	"ai-buddy/internal/tracker/tasks"
)

func TestRoutePolicyAndContracts(t *testing.T) {
	db := testutil.Database(t, migrations.Files)
	if _, err := db.Exec(`INSERT INTO profiles(authentik_id,name,username,email,role,team) VALUES ('i','Intern','i','i@test','intern','backend'),('j','Other','j','j@test','intern','backend'),('m','Mentor','m','m@test','mentor','backend'); INSERT INTO assignments(intern_id,mentor_id,team) VALUES ('i','m','backend')`); err != nil {
		t.Fatal(err)
	}
	repo := postgres.New(db)
	sessions := scs.New()
	sessions.Store = memstore.NewWithCleanupInterval(0)
	login := auth.New(sessions, nil)
	h := New(tasks.New(repo, repo, time.UTC, time.Now), repo, progress.New(repo, repo, time.UTC, time.Now))
	routes := Routes(login, h, "buddy", "ticket")
	cookie := func(id, role string) *http.Cookie {
		t.Helper()
		ctx, err := sessions.Load(context.Background(), "")
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(auth.User{ID: id, Role: role, Team: "backend"})
		sessions.Put(ctx, auth.SessionKeyUser, string(raw))
		token, _, err := sessions.Commit(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Cookie{Name: sessions.Cookie.Name, Value: token}
	}
	intern, mentor := cookie("i", "intern"), cookie("m", "mentor")
	call := func(method, path, body, key, token string, c *http.Cookie, code int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Idempotency-Key", key)
		r.Header.Set("Authorization", token)
		if c != nil {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		routes.ServeHTTP(w, r)
		if w.Code != code {
			t.Fatalf("%s %s: got %d %s want %d", method, path, w.Code, w.Body, code)
		}
		return w
	}
	body := `{"title":"Task","deadline":"2026-12-01","description":null}`
	browser := []struct {
		method, path, body string
		cookie             *http.Cookie
	}{
		{"GET", "/api/tasks", "", intern}, {"POST", "/api/tasks", body, intern},
		{"POST", "/api/tasks/missing/complete", "", intern}, {"GET", "/api/progress", "", intern},
		{"GET", "/api/interns", "", mentor}, {"GET", "/api/interns/i/progress", "", mentor},
	}
	for _, route := range browser {
		call(route.method, route.path, route.body, "key", "", nil, 401)
		wrong := mentor
		if route.cookie == mentor {
			wrong = intern
		}
		call(route.method, route.path, route.body, "key", "", wrong, 403)
		code := 200
		if route.method == "POST" {
			code = 201
			if strings.HasSuffix(route.path, "/complete") {
				code = 404
			}
		}
		call(route.method, route.path, route.body, "key", "", route.cookie, code)
		w := call("DELETE", route.path, "", "", "", route.cookie, 405)
		if w.Header().Get("Allow") == "" {
			t.Fatal("missing Allow")
		}
	}
	internal := []struct {
		method, path, body string
		code               int
	}{
		{"GET", "/internal/interns/i/progress", "", 200},
		{"POST", "/internal/interns/i/tasks", body, 200},
		{"POST", "/internal/interns/i/tasks/missing/complete", "", 404},
		{"GET", "/internal/interns/i/mentor", "", 200},
	}
	for _, route := range internal {
		for _, token := range []string{"", "wrong", "Basic buddy", "Bearer wrong", "Bearer buddy extra"} {
			call(route.method, route.path, route.body, "key", token, nil, 401)
		}
		call(route.method, route.path, route.body, "key", "Bearer buddy", nil, route.code)
		code := 403
		if strings.HasSuffix(route.path, "/mentor") {
			code = 200
		}
		call(route.method, route.path, route.body, "key", "Bearer ticket", nil, code)
	}
	// Unknown intern and known unassigned intern remain distinct.
	call("GET", "/internal/interns/missing/mentor", "", "", "Bearer buddy", nil, 404)
	if w := call("GET", "/internal/interns/j/mentor", "", "", "Bearer ticket", nil, 200); strings.TrimSpace(w.Body.String()) != `{"mentor":null}` {
		t.Fatal(w.Body)
	}
	call("GET", "/api/interns/j/progress", "", "", "", mentor, 404)
	call("HEAD", "/api/tasks", "", "", "", intern, 200)
	call("GET", "/api/missing", "", "", "", nil, 404)
	call("POST", "/api/tasks", `{"title":" Task ","description":" Details ","deadline":" 2026-12-01 "}`, "trimmed", "", intern, 201)
	// Decoder legacy behavior: case-insensitive matching, last duplicate wins.
	w := call("POST", "/api/tasks", `{"TITLE":"first","title":"last","deadline":"2026-12-01"}`, "legacy", "", intern, 201)
	var task tasks.Task
	if err := json.Unmarshal(w.Body.Bytes(), &task); err != nil || task.Title != "last" {
		t.Fatal(task, err)
	}
	call("POST", "/api/tasks/"+task.ID+"/complete", "", "", "", intern, 200)
	for _, bad := range []string{`{"owner":"j"}`, `{} {}`, `{`, strings.Repeat("x", (1<<20)+1)} {
		call("POST", "/api/tasks", bad, "bad", "", intern, 400)
	}
	call("POST", "/api/tasks", body, strings.Repeat("я", 129), "", intern, 400)
	call("POST", "/api/tasks", body, strings.Repeat("я", 128), "", intern, 201)
	r := httptest.NewRequest("POST", "/api/tasks", strings.NewReader(body))
	r.AddCookie(intern)
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	w = httptest.NewRecorder()
	routes.ServeHTTP(w, r)
	if w.Code != 403 || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		t.Fatal(w.Code, w.Header())
	}
}
