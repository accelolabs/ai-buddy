package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	// Project packages.
	"backend/internal/buddy/agent"
	buddyhandler "backend/internal/buddy/handler"
	"backend/internal/ollama"
	"backend/internal/ticket"
	"backend/internal/tracker"
	"backend/internal/tracker/handler"
	"backend/internal/users"
	usershandler "backend/internal/users/handler"
	"backend/internal/users/middleware"
	usersrepository "backend/internal/users/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Each test applies the real migrations in a separate schema.
func testRepository(t *testing.T) (*Repository, context.Context) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}

	admin := stdlib.OpenDB(*cfg)
	t.Cleanup(func() {
		if err := admin.Close(); err != nil {
			t.Error(err)
		}
	})
	schema := fmt.Sprintf("tracker_test_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, `
		CREATE SCHEMA
	`+schema); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()

		if _, err := admin.ExecContext(cleanupCtx, `
			DROP SCHEMA
		`+schema+` CASCADE`); err != nil {
			t.Error(err)
		}
	})

	cfg.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*cfg)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}

	if err := goose.UpContext(ctx, db, "../../../migrations"); err != nil {
		t.Fatal(err)
	}

	return NewRepository(db), ctx
}

func TestTasks(t *testing.T) {
	repo, ctx := testRepository(t)
	if err := users.Seed(ctx, repo.db); err != nil {
		t.Fatal(err)
	}

	intern := users.User{
		ID:   "intern.backend.1",
		Role: users.RoleIntern,
	}
	second := users.User{
		ID:   "intern.backend.2",
		Role: users.RoleIntern,
	}
	mentor := users.User{
		ID:   "mentor.backend",
		Role: users.RoleMentor,
	}
	otherMentor := users.User{
		ID:   "mentor.platform",
		Role: users.RoleMentor,
	}

	for _, user := range []users.User{intern, mentor} {
		got, err := repo.List(ctx, user, intern.ID)
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("empty: %v %v", got, err)
		}
	}

	past := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	future := past.Add(2 * time.Hour)

	for _, value := range []tracker.Task{
		{
			ID:          "z",
			UserID:      intern.ID,
			Description: "Задача 🔑",
			Deadline:    future,
		},
		{
			ID:          "a",
			UserID:      intern.ID,
			Description: "Вторая",
			Deadline:    future,
		},
		{
			ID:          "b",
			UserID:      intern.ID,
			Description: "Просрочена",
			Deadline:    past,
		},
		{
			ID:          "other",
			UserID:      second.ID,
			Description: "Чужая",
			Deadline:    past,
		},
	} {
		got, err := repo.Add(ctx, value)
		if err != nil || got.CreatedAt.IsZero() {
			t.Fatalf("add: %+v %v", got, err)
		}

		value.CreatedAt = got.CreatedAt
		value.Status = "pending"
		if value.Deadline.Before(time.Now()) {
			value.Status = "overdue"
		}

		if !reflect.DeepEqual(got, value) {
			t.Fatalf("round trip: %+v want %+v", got, value)
		}
	}

	if _, err := repo.Add(ctx, tracker.Task{
		ID:       "orphan",
		UserID:   "missing",
		Deadline: future,
	}); err == nil {
		t.Fatal("orphan accepted")
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			UPDATE tasks
			SET created_at='2026-01-01Z'
		`,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			UPDATE tasks
			SET created_at='2026-01-02Z'
			WHERE id='a'
		`,
	); err != nil {
		t.Fatal(err)
	}

	checkList := func(user users.User, id string, ids []string) []tracker.Task {
		t.Helper()
		got, err := repo.List(ctx, user, id)
		if err != nil {
			t.Fatal(err)
		}

		actual := make([]string, 0, len(got))

		for _, v := range got {
			actual = append(actual, v.ID)
		}

		if !reflect.DeepEqual(actual, ids) {
			t.Fatalf("list: %v want %v", actual, ids)
		}

		return got
	}
	checkList(intern, intern.ID, []string{"b", "z", "a"})
	checkList(mentor, intern.ID, []string{"b", "z", "a"})
	// Equal deadline and creation timestamps are ordered by ID.
	if _, err := repo.db.ExecContext(
		ctx,
		`
			UPDATE tasks
			SET created_at='2026-01-01Z'
			WHERE id='a'
		`,
	); err != nil {
		t.Fatal(err)
	}

	checkList(intern, intern.ID, []string{"b", "a", "z"})

	for _, tc := range []struct {
		user users.User
		id   string
	}{
		{intern, second.ID},
		{mentor, "intern.platform.1"},
		{mentor, "missing"},
		{mentor, mentor.ID},
		{otherMentor, intern.ID},
	} {
		if _, err := repo.List(ctx, tc.user, tc.id); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("access: %v", err)
		}
	}

	for _, user := range []users.User{second, mentor} {
		if _, err := repo.Complete(ctx, user, "b"); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("complete access: %v", err)
		}
	}

	if _, err := repo.Complete(ctx, intern, "missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}

	for range 2 {
		got, err := repo.Complete(ctx, intern, "b")
		if err != nil || got.Status != "completed" {
			t.Fatalf("complete overdue: %+v %v", got, err)
		}
	}

	list := checkList(intern, intern.ID, []string{"b", "a", "z"})
	if list[0].Status != "completed" {
		t.Fatal("completed lost priority")
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			UPDATE tasks
			SET deadline=now()-interval '1 day'
			WHERE id='a'
		`,
	); err != nil {
		t.Fatal(err)
	}

	list = checkList(intern, intern.ID, []string{"a", "b", "z"})
	if list[0].Status != "overdue" {
		t.Fatal("status was not recalculated")
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			UPDATE tasks
			SET deadline=now()+interval '1 day'
			WHERE id='a'
		`,
	); err != nil {
		t.Fatal(err)
	}

	list = checkList(intern, intern.ID, []string{"b", "z", "a"})
	if list[2].Status != "pending" {
		t.Fatal("pending was not recalculated")
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			UPDATE mentorships
			SET mentor_id='mentor.platform'
			WHERE intern_id=$1
		`,
		intern.ID,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.List(ctx, mentor, intern.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("old mentor retains access", err)
	}

	checkList(otherMentor, intern.ID, []string{"b", "z", "a"})
	if _, err := repo.db.ExecContext(
		ctx,
		`
			DELETE
			FROM mentorships
			WHERE intern_id=$1
		`,
		intern.ID,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.List(ctx, otherMentor, intern.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("removed assignment retains access", err)
	}
}

func TestTrackerMigration(t *testing.T) {
	repo, ctx := testRepository(t)
	if err := users.Seed(ctx, repo.db); err != nil {
		t.Fatal(err)
	}

	for _, query := range []string{
		`
			INSERT INTO documents (id,path,team,title)
			VALUES ('doc','doc.md','','Wiki')
		`,
		`
			INSERT INTO chunks (document_id,ordinal,anchor,text)
			VALUES ('doc',0,'','Text')
		`,
		`
			INSERT INTO tickets (id,user_id,title,description)
			VALUES ('ticket','intern.backend.1','Title','Text')
		`,
		`
			INSERT INTO tasks (id,user_id,description,deadline)
			VALUES ('task','intern.backend.1','Text',now())
		`,
	} {
		if _, err := repo.db.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}

	for range 2 {
		if err := goose.DownToContext(ctx, repo.db, "../../../migrations", 3); err != nil {
			t.Fatal(err)
		}

		for table, want := range map[string]int{
			"users":       6,
			"mentorships": 4,
			"documents":   1,
			"chunks":      1,
			"tickets":     1,
		} {
			var count int
			if err := repo.db.QueryRowContext(ctx, `
				SELECT count(*)
				FROM
			`+table).Scan(&count); err != nil || count != want {
				t.Fatalf("preserve %s: %d %v", table, count, err)
			}
		}

		var exists bool
		if err := repo.db.QueryRowContext(
			ctx,
			`
				SELECT to_regclass(current_schema()||'.tasks') IS NOT NULL
			`,
		).Scan(&exists); err != nil ||
			exists {
			t.Fatalf("tasks survived down: %v", err)
		}

		if err := goose.UpContext(ctx, repo.db, "../../../migrations"); err != nil {
			t.Fatal(err)
		}

		var count int
		if err := repo.db.QueryRowContext(
			ctx,
			`
				SELECT count(*)
				FROM tasks
			`,
		).Scan(&count); err != nil || count != 0 {
			t.Fatalf("reapply: %d %v", count, err)
		}
	}
}

func TestHTTP(t *testing.T) {
	repo, ctx := testRepository(t)
	if err := users.Seed(ctx, repo.db); err != nil {
		t.Fatal(err)
	}

	ur := usersrepository.NewRepository(repo.db)
	h := handler.NewHandler(repo)
	uh := usershandler.NewHandler(ur)
	selected := middleware.Middleware(ur)
	mux := http.NewServeMux()
	mux.Handle("POST /api/tracker/tasks", selected(http.HandlerFunc(h.Create)))
	mux.Handle("GET /api/tracker/tasks", selected(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/tracker/interns/{internID}/tasks", selected(http.HandlerFunc(h.ListIntern)))
	mux.Handle("POST /api/tracker/tasks/{id}/complete", selected(http.HandlerFunc(h.Complete)))
	mux.Handle("GET /api/users/interns", selected(http.HandlerFunc(uh.ListInterns)))
	request := func(method, path, user, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if user != "" {
			r.AddCookie(&http.Cookie{
				Name:  "demo_user",
				Value: user,
			})
		}

		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s %s: %d %s", method, path, user, w.Code, w.Body)
		}

		return w
	}
	intern := "intern.backend.1"
	mentor := "mentor.backend"
	valid := "{\"description\":\"  Задача 🔑 " +
		" \",\"deadline\":\"2099-01-01T12:00:00+03:00\",\"user_id\":\"intern.platf" +
		"orm.1\"}"

	for _, user := range []string{"", "missing"} {
		request("POST", "/api/tracker/tasks", user, valid, 401)
		request("GET", "/api/tracker/tasks", user, "", 401)
		request("GET", "/api/tracker/interns/"+intern+"/tasks", user, "", 401)
		request("POST", "/api/tracker/tasks/missing/complete", user, "", 401)
		request("GET", "/api/users/interns", user, "", 401)
	}

	request("POST", "/api/tracker/tasks", mentor, valid, 403)
	request("POST", "/api/tracker/tasks/missing/complete", mentor, "", 403)
	request("GET", "/api/tracker/tasks", mentor, "", 403)
	request("GET", "/api/tracker/interns/"+intern+"/tasks", intern, "", 403)
	request("GET", "/api/users/interns", intern, "", 403)

	for _, body := range []string{
		"",
		`null`,
		`{}`,
		`{"description":" ","deadline":"2099-01-01T00:00:00Z"}`,
		`{"description":"x"}`,
		`{"description":"x","deadline":"2099-01-01T00:00:00"}`,
		`{"description":"x","deadline":"bad"}`,
		`{"description":"x","deadline":123}`,
		valid + ` {}`,
		valid + ` rubbish`,
		`{"description":"` + strings.Repeat(
			"x",
			1<<20,
		) + `","deadline":"2099-01-01T00:00:00Z"}`,
		valid + strings.Repeat(
			" ",
			1<<20,
		),
	} {
		request("POST", "/api/tracker/tasks", intern, body, 400)
	}

	for _, path := range []string{"/api/tracker/tasks", "/api/tracker/interns/" + intern + "/tasks"} {
		user := intern
		if strings.Contains(path, "interns") {
			user = mentor
		}

		if w := request("GET", path, user, "", 200); w.Body.String() != "[]\n" {
			t.Fatal(w.Body)
		}
	}

	for _, id := range []string{"missing", "intern.platform.1", mentor} {
		request("GET", "/api/tracker/interns/"+id+"/tasks", mentor, "", 404)
	}

	var created taskResponseForTest
	w := request("POST", "/api/tracker/tasks", intern, valid, 201)
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	if created.ID == "" || created.UserID != intern || created.Description != "Задача 🔑" ||
		created.Status != "pending" ||
		created.CreatedAt.IsZero() ||
		created.Deadline.Format(time.RFC3339) != "2099-01-01T12:00:00+03:00" {
		t.Fatalf("create: %+v", created)
	}

	for _, tc := range []struct {
		user, path string
	}{{intern, "/api/tracker/tasks"}, {mentor, "/api/tracker/interns/" + intern + "/tasks"}} {
		var list []taskResponseForTest
		if err := json.Unmarshal(request("GET", tc.path, tc.user, "", 200).Body.Bytes(), &list); err != nil {
			t.Fatal(err)
		}

		if len(list) != 1 || list[0].ID != created.ID || !list[0].Deadline.Equal(created.Deadline) {
			t.Fatalf("list: %+v", list)
		}
	}

	path := "/api/tracker/tasks/" + created.ID + "/complete"
	request("POST", path, intern, `{}`, 400)
	request("POST", path, "intern.backend.2", "", 404)
	request("POST", "/api/tracker/tasks/missing/complete", intern, "", 404)

	var first string

	for i := range 2 {
		w := request("POST", path, intern, "", 200)

		var got taskResponseForTest
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}

		if got.ID != created.ID || got.Status != "completed" || !got.CreatedAt.Equal(created.CreatedAt) {
			t.Fatalf("complete: %+v", got)
		}

		if i == 0 {
			first = w.Body.String()
		} else if first != w.Body.String() {
			t.Fatal("repeat changed response")
		}
	}

	w = request(
		"POST",
		"/api/tracker/tasks",
		intern,
		`{"description":"Past","deadline":"2000-01-01T00:00:00Z"}`,
		201,
	)
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.Status != "overdue" {
		t.Fatalf("past: %+v %v", created, err)
	}
	// Only the current mentor's interns are exposed, including an empty assignment set.
	for _, user := range []string{mentor, "mentor.platform"} {
		var list []users.User
		w = request("GET", "/api/users/interns", user, "", 200)
		if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list) != 2 {
			t.Fatalf("interns: %s %v", w.Body, err)
		}

		for _, v := range list {
			if !strings.HasPrefix(v.ID, "intern."+strings.TrimPrefix(user, "mentor.")+".") {
				t.Fatal("foreign intern", v)
			}
		}
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			UPDATE mentorships
			SET mentor_id='mentor.platform'
			WHERE intern_id=$1
		`,
		intern,
	); err != nil {
		t.Fatal(err)
	}

	request("GET", "/api/tracker/interns/"+intern+"/tasks", mentor, "", 404)
	request("GET", "/api/tracker/interns/"+intern+"/tasks", "mentor.platform", "", 200)
	if _, err := repo.db.ExecContext(
		ctx,
		`
			DELETE
			FROM mentorships
			WHERE mentor_id=$1
		`,
		mentor,
	); err != nil {
		t.Fatal(err)
	}

	if w := request("GET", "/api/users/interns", mentor, "", 200); w.Body.String() != "[]\n" {
		t.Fatal(w.Body)
	}
}

type taskResponseForTest struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Description string    `json:"description"`
	Deadline    time.Time `json:"deadline"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

func TestTaskSeed(t *testing.T) {
	repo, ctx := testRepository(t)
	if err := users.Seed(ctx, repo.db); err != nil {
		t.Fatal(err)
	}

	if err := ticket.Seed(ctx, repo.db); err != nil {
		t.Fatal(err)
	}

	if err := tracker.Seed(ctx, repo.db); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := repo.db.QueryRowContext(
		ctx,
		`
			SELECT count(*)
			FROM tasks
		`,
	).Scan(&count); err != nil || count != 12 {
		t.Fatalf("count: %d %v", count, err)
	}

	for _, id := range []string{"intern.backend.1", "intern.backend.2", "intern.platform.1", "intern.platform.2"} {
		values, err := repo.List(ctx, users.User{
			ID:   id,
			Role: users.RoleIntern,
		}, id)
		if err != nil || len(values) != 3 {
			t.Fatalf("author: %+v %v", values, err)
		}

		for i, value := range values {
			if value.UserID != id || value.Description == "" || value.CreatedAt.IsZero() ||
				!strings.HasPrefix(
					value.ID,
					"demo.task."+id+".",
				) ||
				value.Status != []string{"completed", "overdue", "pending"}[i] {
				t.Fatal(value)
			}

			if delta := value.Deadline.Sub(value.CreatedAt); delta != []time.Duration{
				-48 * time.Hour,
				-24 * time.Hour,
				72 * time.Hour,
			}[i] {
				t.Fatalf("deadline offset: %v", delta)
			}
		}
	}

	snapshotSQL := `
		SELECT json_agg(t
			ORDER BY id)::text
		FROM tasks t
	`

	var before, after string
	if err := repo.db.QueryRowContext(ctx, snapshotSQL).Scan(&before); err != nil {
		t.Fatal(err)
	}

	if err := tracker.Seed(ctx, repo.db); err != nil {
		t.Fatal(err)
	}

	if err := repo.db.QueryRowContext(ctx, snapshotSQL).Scan(&after); err != nil || before != after {
		t.Fatalf("rerun changed data: %v", err)
	}

	if _, err := repo.Complete(
		ctx,
		users.User{
			ID:   "intern.backend.1",
			Role: users.RoleIntern,
		},
		"demo.task.intern.backend.1.1",
	); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			UPDATE tasks
			SET description='Изменено', deadline='2020-01-01Z', created_at='2019-01-01Z'
			WHERE id='demo.task.intern.backend.1.1'
		`,
	); err != nil {
		t.Fatal(err)
	}

	if err := repo.db.QueryRowContext(ctx, snapshotSQL).Scan(&before); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := users.Seed(ctx, repo.db); err != nil {
			t.Fatal(err)
		}

		if err := ticket.Seed(ctx, repo.db); err != nil {
			t.Fatal(err)
		}

		if err := tracker.Seed(ctx, repo.db); err != nil {
			t.Fatal(err)
		}
	}

	if err := repo.db.QueryRowContext(ctx, snapshotSQL).Scan(&after); err != nil || before != after {
		t.Fatalf("overwrote edits/completion: %v", err)
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			DELETE
			FROM tasks
		`,
	); err != nil {
		t.Fatal(err)
	}
	// A late SQL failure must roll back tasks and preserve prior seed stages.
	if _, err := repo.db.ExecContext(
		ctx,
		`
			ALTER TABLE tasks ADD CONSTRAINT seed_failure CHECK (user_id <> 'intern.platform.2')
		`,
	); err != nil {
		t.Fatal(err)
	}

	if err := tracker.Seed(ctx, repo.db); err == nil {
		t.Fatal("constraint failure ignored")
	}

	if err := repo.db.QueryRowContext(
		ctx,
		`
			SELECT count(*)
			FROM tasks
		`,
	).Scan(&count); err != nil || count != 0 {
		t.Fatalf("partial tasks: %d %v", count, err)
	}

	if err := repo.db.QueryRowContext(
		ctx,
		`
			SELECT count(*)
			FROM tickets
		`,
	).Scan(&count); err != nil || count != 8 {
		t.Fatalf("earlier tickets lost: %d %v", count, err)
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			ALTER TABLE tasks DROP CONSTRAINT seed_failure
		`,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			DELETE
			FROM tickets
			WHERE user_id='intern.platform.2'
		`,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			DELETE
			FROM users
			WHERE id='intern.platform.2'
		`,
	); err != nil {
		t.Fatal(err)
	}

	if err := tracker.Seed(ctx, repo.db); err == nil {
		t.Fatal("missing author accepted")
	}

	if err := repo.db.QueryRowContext(
		ctx,
		`
			SELECT count(*)
			FROM tasks
		`,
	).Scan(&count); err != nil || count != 0 {
		t.Fatalf("partial tasks: %d %v", count, err)
	}
}

type taskModel struct{}

func (taskModel) Chat(
	_ context.Context,
	messages []ollama.Message,
	_ []ollama.Tool,
	_ func(string) error,
) (ollama.Response, error) {
	if messages[len(messages)-1].Role == "tool" {
		return ollama.Response{Message: ollama.Message{Content: "Подтвердите кнопкой"}}, nil
	}

	return ollama.Response{
		Message: ollama.Message{
			Calls: []ollama.Call{{
				Function: ollama.Function{
					Name:      "prepare_task",
					Arguments: []byte(`{"description":"Черновик","deadline":"2030-01-01T12:00:00+03:00"}`),
				},
			}},
		},
	}, nil
}

func TestPrepareTaskSSEAndCreate(t *testing.T) {
	repo, ctx := testRepository(t)
	if err := users.Seed(ctx, repo.db); err != nil {
		t.Fatal(err)
	}

	a, err := agent.NewAgent(taskModel{}, nil, "../../../prompts/buddy.md", nil)
	if err != nil {
		t.Fatal(err)
	}

	selected := middleware.Middleware(usersrepository.NewRepository(repo.db))
	mux := http.NewServeMux()
	mux.Handle(
		"POST /api/buddy/messages",
		selected(http.HandlerFunc(buddyhandler.NewHandler(a, 6).Send)),
	)
	mux.Handle("POST /api/tracker/tasks", selected(http.HandlerFunc(handler.NewHandler(repo).Create)))
	request := func(path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", path, strings.NewReader(body)).WithContext(ctx)
		r.AddCookie(&http.Cookie{
			Name:  "demo_user",
			Value: "intern.backend.1",
		})
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		return w
	}
	response := request(
		"/api/buddy/messages",
		`{"messages":[{"role":"user","content":"Подготовь задачу"}]}`,
	)
	if response.Code != 200 {
		t.Fatalf("SSE: %d %s", response.Code, response.Body)
	}

	var draft struct {
		Description string    `json:"description"`
		Deadline    time.Time `json:"deadline"`
	}
	lines := strings.Split(response.Body.String(), "\n")
	found := false

	for i, line := range lines {
		if line == "event: task" {
			if err := json.Unmarshal([]byte(strings.TrimPrefix(lines[i+1], "data: ")), &draft); err != nil {
				t.Fatal(err)
			}

			found = true
		}
	}

	if !found || draft.Description != "Черновик" ||
		draft.Deadline.Format(time.RFC3339) != "2030-01-01T12:00:00+03:00" {
		t.Fatalf("draft: %+v %s", draft, response.Body)
	}

	var count int
	if err := repo.db.QueryRowContext(
		ctx,
		`
			SELECT count(*)
			FROM tasks
		`,
	).Scan(&count); err != nil || count != 0 {
		t.Fatalf("prepare wrote tasks: %d %v", count, err)
	}

	response = request(
		"/api/tracker/tasks",
		"{\"description\":\" Окончательное описание"+
			" \",\"deadline\":\"2031-02-03T10:00:00Z\",\"user_id\":\"intern.platform.1"+
			"\"}",
	)
	if response.Code != 201 {
		t.Fatalf("POST: %d %s", response.Code, response.Body)
	}

	values, err := repo.List(ctx, users.User{
		ID:   "intern.backend.1",
		Role: users.RoleIntern,
	}, "intern.backend.1")
	if err != nil || len(values) != 1 {
		t.Fatalf("stored: %+v %v", values, err)
	}

	value := values[0]
	if value.Description != "Окончательное описание" ||
		value.Deadline.UTC().Format(time.RFC3339) != "2031-02-03T10:00:00Z" ||
		value.UserID != "intern.backend.1" {
		t.Fatal(value)
	}
}
