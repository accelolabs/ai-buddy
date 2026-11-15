package app

import (
	"ai-buddy/internal/buddy"
	buddypostgres "ai-buddy/internal/buddy/postgres"
	"ai-buddy/internal/config"
	"ai-buddy/internal/migrations"
	"ai-buddy/internal/seed"
	"ai-buddy/internal/testutil"
	"ai-buddy/internal/users"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnifiedAPI(t *testing.T) {
	db := testutil.Database(t, migrations.Files)
	ctx := context.Background()
	if err := seed.Run(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE profiles SET goals='keep me' WHERE id='intern.backend.1'`); err != nil {
		t.Fatal(err)
	}
	if err := seed.Run(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`INSERT INTO assignments(intern_id,mentor_id,team) VALUES('intern.platform.2','mentor.backend','platform')`,
		`INSERT INTO assignments(intern_id,mentor_id,team) VALUES('mentor.platform','intern.platform.1','platform')`,
		`INSERT INTO chats(id,owner_id) VALUES('invalid','missing')`,
		`INSERT INTO tickets(id,sender_id,recipient_id,question,description,idempotency_key) VALUES('bad','missing','mentor.backend','q','d','bad')`,
	} {
		if _, err := db.Exec(query); err == nil {
			t.Fatal("accepted invalid relation", query)
		}
	}

	c := config.Config{Tracker: config.Tracker{Timezone: "Europe/Moscow"}, Ollama: config.Ollama{URL: "http://127.0.0.1:1"}}
	a, err := New(ctx, db, c)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	call := func(method, path, id, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if id != "" {
			r.AddCookie(&http.Cookie{Name: "demo_user", Value: id})
		}
		r.Header.Set("Idempotency-Key", "test")
		w := httptest.NewRecorder()
		a.Handler.ServeHTTP(w, r)
		return w
	}
	w := call("GET", "/api/users", "", "")
	var list struct {
		Users []users.User `json:"users"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &list); err != nil || w.Code != 200 || len(list.Users) != 6 {
		t.Fatal(w, err)
	}
	for i := 1; i < len(list.Users); i++ {
		if list.Users[i-1].ID >= list.Users[i].ID {
			t.Fatal("unsorted users")
		}
	}
	for _, tc := range []struct {
		method, path, id, body string
		status                 int
	}{
		{"GET", "/health/ready", "", "", 204}, {"GET", "/api/me", "", "", 400}, {"GET", "/api/me", "unknown", "", 400},
		{"GET", "/api/me", "intern.backend.1", "", 200}, {"GET", "/api/tasks", "intern.backend.1", "", 405},
		{"GET", "/api/tasks", "mentor.backend", "", 405}, {"GET", "/api/interns", "intern.backend.1", "", 403},
		{"GET", "/api/interns", "mentor.backend", "", 200}, {"GET", "/api/interns/intern.platform.1/progress", "mentor.backend", "", 404},
		{"GET", "/api/progress", "intern.platform.2", "", 200}, {"GET", "/api/tickets/incoming", "mentor.backend", "", 200},
		{"GET", "/api/documents", "intern.backend.1", "", 200}, {"POST", "/api/search", "intern.backend.1", `{"query":"test"}`, 404},
		{"POST", "/api/chats", "mentor.backend", `{}`, 403}, {"GET", "/api/auth/me", "intern.backend.1", "", 404},
		{"GET", "/internal/interns/intern.backend.1/progress", "intern.backend.1", "", 404},
		{"POST", "/api/tickets", "intern.platform.2", `{"question":"q","description":"d"}`, 422},
	} {
		w = call(tc.method, tc.path, tc.id, tc.body)
		if w.Code != tc.status {
			t.Fatalf("%s %s as %s: %d %s", tc.method, tc.path, tc.id, w.Code, w.Body)
		}
	}
	for _, id := range []string{"", "intern.backend.1", "mentor.backend"} {
		w = call("GET", "/api/tasks", id, "")
		if w.Code != 405 || w.Header().Get("Allow") != "POST" || !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
			t.Fatal(w)
		}
		for _, method := range []string{"GET", "POST"} {
			w = call(method, "/api/search", id, `{"query":"test"}`)
			if w.Code != 404 || !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
				t.Fatal(w)
			}
		}
	}
	w = call("POST", "/api/tasks", "intern.backend.1", `{"title":"HTTP task","description":"kept","deadline":"2026-12-01"}`)
	var task struct {
		ID string `json:"id"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &task); err != nil || w.Code != 201 || task.ID == "" {
		t.Fatal(w, err)
	}
	w = call("POST", "/api/tasks/"+task.ID+"/complete", "intern.backend.1", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"status":"completed"`) {
		t.Fatal(w)
	}
	w = call("GET", "/api/progress", "intern.backend.1", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), task.ID) {
		t.Fatal("progress must include tasks", w)
	}
	w = call("POST", "/api/tickets", "intern.backend.1", `{"question":"HTTP ticket","description":"kept"}`)
	if w.Code != 201 {
		t.Fatal(w)
	}
	for _, path := range []string{"/api/tasks", "/api/tasks/" + task.ID + "/complete", "/api/tickets"} {
		if w = call("POST", path, "mentor.backend", `{}`); w.Code != 403 {
			t.Fatal(w)
		}
	}

	w = call("GET", "/api/me", "intern.backend.1", "")
	if !strings.Contains(w.Body.String(), `"goals":"keep me"`) {
		t.Fatal("seed overwrote state", w)
	}
	w = call("GET", "/api/interns", "mentor.backend", "")
	if strings.Contains(w.Body.String(), "platform") {
		t.Fatal("cross-team interns", w)
	}
	w = call("GET", "/api/tickets/incoming", "mentor.backend", "")
	if strings.Contains(w.Body.String(), "platform") {
		t.Fatal("cross-team tickets", w)
	}
	repo := buddypostgres.New(db)
	chat, err := repo.CreateChat(ctx, "intern.backend.1")
	if err != nil {
		t.Fatal(err)
	}
	run, _, err := repo.Start(ctx, "intern.backend.1", chat.ID, "replay", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.Event(ctx, run.ID, buddy.EventText, "saved"); err != nil {
		t.Fatal(err)
	}
	if err = repo.Finish(ctx, run.ID, ""); err != nil {
		t.Fatal(err)
	}
	w = call("GET", "/api/runs/"+run.ID+"/events", "intern.backend.1", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "saved") || w.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatal(w)
	}
	w = call("GET", "/api/runs/"+run.ID+"/events", "intern.backend.2", "")
	if w.Code != 404 {
		t.Fatal(w)
	}
	// A restart preserves chats and turns an interrupted generation into a persisted error.
	interrupted, err := repo.CreateChat(ctx, "intern.backend.1")
	if err != nil {
		t.Fatal(err)
	}
	pending, _, err := repo.Start(ctx, "intern.backend.1", interrupted.ID, "recover", "pending")
	if err != nil {
		t.Fatal(err)
	}
	a.Close()
	second, err := New(ctx, db, c)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	saved, err := repo.Run(ctx, "intern.backend.1", pending.ID)
	if err != nil || saved.Status != "error" {
		t.Fatal(fmt.Sprint(saved), err)
	}
}
