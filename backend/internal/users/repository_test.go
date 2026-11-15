package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"reflect"
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
	schema := fmt.Sprintf("users_test_%d", time.Now().UnixNano())
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
	if err := goose.UpContext(ctx, db, "../../migrations"); err != nil {
		t.Fatal(err)
	}

	return NewRepository(db), ctx
}

func TestRepository(t *testing.T) {
	repo, ctx := testRepository(t)
	list, err := repo.List(ctx)
	if err != nil || list == nil || len(list) != 0 {
		t.Fatalf("empty List() = %v, %v", list, err)
	}
	if _, err := repo.Get(ctx, "missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Get(missing) error = %v, want sql.ErrNoRows", err)
	}

	first := User{ID: "a", Name: "Стажёр", Role: RoleIntern, Team: "backend"}
	second := User{ID: "b", Name: "Ментор", Role: RoleMentor, Team: "platform"}
	for _, user := range []User{second, first} {
		if err := repo.Add(ctx, user); err != nil {
			t.Fatal(err)
		}
		got, err := repo.Get(ctx, user.ID)
		if err != nil || got != user {
			t.Fatalf("Get(%q) = %+v, %v; want %+v", user.ID, got, err, user)
		}
	}
	if err := repo.Add(ctx, first); err == nil {
		t.Fatal("duplicate ID was accepted")
	}
	list, err = repo.List(ctx)
	if err != nil || !reflect.DeepEqual(list, []User{first, second}) {
		t.Fatalf("List() = %+v, %v; want both users ordered by ID", list, err)
	}

	if err := repo.Add(ctx, User{ID: "bad-role", Name: "User", Role: "admin", Team: "backend"}); err == nil {
		t.Fatal("invalid role was accepted")
	}

	if err := goose.DownToContext(ctx, repo.db, "../../migrations", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.List(ctx); err == nil {
		t.Fatal("users table still exists after migration rollback")
	}
	if err := goose.UpContext(ctx, repo.db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
}

func TestSeed(t *testing.T) {
	repo, ctx := testRepository(t)
	// Fail halfway through the seed to check that no partial dataset remains.
	if _, err := repo.db.ExecContext(ctx,
		`ALTER TABLE users ADD CONSTRAINT reject_platform CHECK (team <> 'platform')`,
	); err != nil {
		t.Fatal(err)
	}
	if err := repo.Seed(ctx); err == nil {
		t.Fatal("Seed() succeeded despite rejected platform users")
	}
	list, err := repo.List(ctx)
	if err != nil || len(list) != 0 {
		t.Fatalf("failed seed left users: %+v, %v", list, err)
	}
	if _, err := repo.db.ExecContext(ctx, `ALTER TABLE users DROP CONSTRAINT reject_platform`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := repo.Seed(ctx); err != nil {
			t.Fatal(err)
		}
	}
	list, err = repo.List(ctx)
	if err != nil || len(list) != 6 {
		t.Fatalf("List() after two seeds = %+v, %v; want 6 users", list, err)
	}
	counts := make(map[string]int)
	for _, user := range list {
		counts[user.Team+"/"+user.Role]++
	}
	wantCounts := map[string]int{
		"backend/intern": 2, "backend/mentor": 1,
		"platform/intern": 2, "platform/mentor": 1,
	}
	if !reflect.DeepEqual(counts, wantCounts) {
		t.Fatalf("seeded teams and roles = %v, want %v", counts, wantCounts)
	}

	if _, err := repo.db.ExecContext(ctx,
		`UPDATE users SET name = 'Сохранённое имя' WHERE id = 'intern.backend.1'`,
	); err != nil {
		t.Fatal(err)
	}
	extra := User{ID: "extra", Name: "Другой пользователь", Role: RoleIntern, Team: "backend"}
	if err := repo.Add(ctx, extra); err != nil {
		t.Fatal(err)
	}
	if err := repo.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	user, err := repo.Get(ctx, "intern.backend.1")
	if err != nil || user.Name != "Сохранённое имя" {
		t.Fatalf("seed changed existing user: %+v, %v", user, err)
	}
	if user, err := repo.Get(ctx, extra.ID); err != nil || user != extra {
		t.Fatalf("seed changed unrelated user: %+v, %v", user, err)
	}
}
