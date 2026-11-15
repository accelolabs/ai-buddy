package repository

import (
	"backend/internal/ticket"
	"backend/internal/ticket/handler"
	"backend/internal/users"
	"backend/internal/users/middleware"
	usersrepository "backend/internal/users/repository"
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
	schema := fmt.Sprintf("ticket_test_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := admin.ExecContext(cleanupCtx, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
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

func TestTickets(t *testing.T) {
	repo, ctx := testRepository(t)
	if err := users.Seed(ctx, repo.db.(*sql.DB)); err != nil {
		t.Fatal(err)
	}
	intern := users.User{ID: "intern.backend.1", Role: users.RoleIntern}
	second := users.User{ID: "intern.backend.2", Role: users.RoleIntern}
	other := users.User{ID: "intern.platform.1", Role: users.RoleIntern}
	mentor := users.User{ID: "mentor.backend", Role: users.RoleMentor}
	emptyMentor := users.User{ID: "unassigned", Role: users.RoleMentor}
	for _, user := range []users.User{intern, mentor, emptyMentor} {
		got, err := repo.List(ctx, user)
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("empty list: %v %v", got, err)
		}
	}
	values := []ticket.Ticket{
		{ID: "a", UserID: intern.ID, Title: "Доступ", Description: "Нет доступа 🔑"},
		{ID: "z", UserID: intern.ID, Title: "Второй", Description: "Описание"},
		{ID: "b", UserID: second.ID, Title: "Другой стажёр", Description: "Описание"},
		{ID: "c", UserID: other.ID, Title: "Чужой", Description: "Описание"},
	}
	for i, value := range values {
		added, err := repo.Add(ctx, value)
		if err != nil || added.CreatedAt.IsZero() {
			t.Fatalf("add: %+v %v", added, err)
		}
		value.CreatedAt = added.CreatedAt
		if value != added {
			t.Fatalf("changed fields: %+v", added)
		}
		got, err := repo.Get(ctx, users.User{ID: value.UserID, Role: users.RoleIntern}, value.ID)
		if err != nil || got != added {
			t.Fatalf("round trip: %+v %v", got, err)
		}
		values[i] = added
	}
	// Force equal timestamps for the ID tie-break, then a newer ticket with a smaller ID.
	if _, err := repo.db.(*sql.DB).ExecContext(ctx, `UPDATE tickets SET created_at = '2026-01-01Z'`); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.db.(*sql.DB).ExecContext(ctx, `UPDATE tickets SET created_at = '2026-01-02Z' WHERE id = 'a'`); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		user users.User
		ids  []string
	}{
		{intern, []string{"a", "z"}},
		{second, []string{"b"}},
		{other, []string{"c"}},
		{mentor, []string{"a", "z", "b"}},
		{emptyMentor, []string{}},
	} {
		list, err := repo.List(ctx, tc.user)
		if err != nil {
			t.Fatal(err)
		}
		ids := make([]string, 0, len(list))
		for _, value := range list {
			ids = append(ids, value.ID)
		}
		if !reflect.DeepEqual(ids, tc.ids) {
			t.Fatalf("list %s: %v want %v", tc.user.ID, ids, tc.ids)
		}
		for _, value := range values {
			_, err := repo.Get(ctx, tc.user, value.ID)
			accessible := false
			for _, id := range tc.ids {
				if id == value.ID {
					accessible = true
				}
			}
			if accessible && err != nil || !accessible && !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("access %s/%s: %v", tc.user.ID, value.ID, err)
			}
		}
		if _, err := repo.Get(ctx, tc.user, "missing"); !errors.Is(err, sql.ErrNoRows) {
			t.Fatal(err)
		}
	}
	if _, err := repo.Add(ctx, values[0]); err == nil {
		t.Fatal("duplicate accepted")
	}
	if _, err := repo.Add(ctx, ticket.Ticket{ID: "orphan", UserID: "missing"}); err == nil {
		t.Fatal("orphan accepted")
	}
	// Access follows current assignments, independently of team membership.
	if _, err := repo.db.(*sql.DB).ExecContext(ctx, `UPDATE mentorships SET mentor_id='mentor.platform' WHERE intern_id=$1`, intern.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, mentor, "a"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("old mentor access: %v", err)
	}
	if _, err := repo.Get(ctx, users.User{ID: "mentor.platform", Role: users.RoleMentor}, "a"); err != nil {
		t.Fatal(err)
	}
	if err := goose.DownToContext(ctx, repo.db.(*sql.DB), "../../../migrations", 3); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.List(ctx, intern); err == nil {
		t.Fatal("table survived rollback")
	}
	var count int
	if err := repo.db.(*sql.DB).QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&count); err != nil || count != 6 {
		t.Fatalf("rollback changed users: %d %v", count, err)
	}
	if err := goose.UpContext(ctx, repo.db.(*sql.DB), "../../../migrations"); err != nil {
		t.Fatal(err)
	}
	if list, err := repo.List(ctx, intern); err != nil || len(list) != 0 {
		t.Fatalf("reapply: %v %v", list, err)
	}
}

func TestHTTPAccess(t *testing.T) {
	repo, ctx := testRepository(t)
	if err := users.Seed(ctx, repo.db.(*sql.DB)); err != nil {
		t.Fatal(err)
	}
	h := handler.NewHandler(repo)
	selected := middleware.Middleware(usersrepository.NewRepository(repo.db.(*sql.DB)))
	mux := http.NewServeMux()

	mux.Handle("GET /api/tickets", selected(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/tickets/{id}", selected(http.HandlerFunc(h.Get)))
	request := func(method, path, user, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if user != "" {
			r.AddCookie(&http.Cookie{Name: "demo_user", Value: user})
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s %s: %d %s", method, path, user, w.Code, w.Body)
		}
		return w
	}
	for _, user := range []string{"", "missing"} {
		request("POST", "/api/tickets", user, `{"title":"x","description":"y"}`, 405)
		request("GET", "/api/tickets", user, "", 401)
		request("GET", "/api/tickets/missing", user, "", 401)
	}
	request("POST", "/api/tickets", "mentor.backend", `{"title":"x","description":"y"}`, 405)
	if _, err := repo.Add(ctx, ticket.Ticket{ID: "created", UserID: "intern.backend.1", Title: "Помогите", Description: "Нет доступа 🔑"}); err != nil {
		t.Fatal(err)
	}
	w := request("GET", "/api/tickets/created", "intern.backend.1", "", 200)
	var created ticketResponseForTest
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	for _, user := range []string{"intern.backend.1", "mentor.backend"} {
		got := request("GET", "/api/tickets/"+created.ID, user, "", 200)
		if got.Body.String() != w.Body.String() {
			t.Fatalf("different format: %s", got.Body)
		}
		list := request("GET", "/api/tickets", user, "", 200)
		if list.Body.String() != "["+strings.TrimSpace(w.Body.String())+"]\n" {
			t.Fatalf("list: %s", list.Body)
		}
	}
	for _, user := range []string{"intern.backend.2", "intern.platform.1", "mentor.platform"} {
		request("GET", "/api/tickets/"+created.ID, user, "", 404)
		list := request("GET", "/api/tickets", user, "", 200)
		if list.Body.String() != "[]\n" {
			t.Fatalf("leaked list: %s", list.Body)
		}
	}
	request("GET", "/api/tickets/missing", "intern.backend.1", "", 404)
}

type ticketResponseForTest struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
