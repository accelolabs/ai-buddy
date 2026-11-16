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

	// Project packages.
	"backend/internal/users"

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

func TestRepository(t *testing.T) {
	repo, ctx := testRepository(t)
	list, err := repo.List(ctx)
	if err != nil || list == nil || len(list) != 0 {
		t.Fatalf("empty List() = %v, %v", list, err)
	}

	if _, err := repo.Get(ctx, "missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Get(missing) error = %v, want sql.ErrNoRows", err)
	}

	first := users.User{
		ID:   "a",
		Name: "Стажёр",
		Role: users.RoleIntern,
		Team: "backend",
	}
	second := users.User{
		ID:   "b",
		Name: "Ментор",
		Role: users.RoleMentor,
		Team: "platform",
	}

	for _, user := range []users.User{second, first} {
		if err := insertUser(ctx, repo.db, user); err != nil {
			t.Fatal(err)
		}

		got, err := repo.Get(ctx, user.ID)
		if err != nil || got != user {
			t.Fatalf("Get(%q) = %+v, %v; want %+v", user.ID, got, err, user)
		}
	}

	if err := insertUser(ctx, repo.db, first); err == nil {
		t.Fatal("duplicate ID was accepted")
	}

	list, err = repo.List(ctx)
	if err != nil || !reflect.DeepEqual(list, []users.User{first, second}) {
		t.Fatalf("List() = %+v, %v; want both users ordered by ID", list, err)
	}

	if err := insertUser(ctx, repo.db, users.User{
		ID:   "bad-role",
		Name: "User",
		Role: "admin",
		Team: "backend",
	}); err == nil {
		t.Fatal("invalid role was accepted")
	}

	if err := goose.DownToContext(ctx, repo.db, "../../../migrations", 0); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.List(ctx); err == nil {
		t.Fatal("users table still exists after migration rollback")
	}

	if err := goose.UpContext(ctx, repo.db, "../../../migrations"); err != nil {
		t.Fatal(err)
	}
}

func TestSeed(t *testing.T) {
	repo, ctx := testRepository(t)
	// Fail halfway through the seed to check that no partial dataset remains.
	if _, err := repo.db.ExecContext(
		ctx,
		`
			ALTER TABLE users ADD CONSTRAINT reject_platform CHECK (team <> 'platform')
		`,
	); err != nil {
		t.Fatal(err)
	}

	if err := users.Seed(ctx, repo.db); err == nil {
		t.Fatal("Seed() succeeded despite rejected platform users")
	}

	list, err := repo.List(ctx)
	if err != nil || len(list) != 0 {
		t.Fatalf("failed seed left users: %+v, %v", list, err)
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			ALTER TABLE users DROP CONSTRAINT reject_platform
		`,
	); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := users.Seed(ctx, repo.db); err != nil {
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
		"backend/intern":  2,
		"backend/mentor":  1,
		"platform/intern": 2,
		"platform/mentor": 1,
	}
	if !reflect.DeepEqual(counts, wantCounts) {
		t.Fatalf("seeded teams and roles = %v, want %v", counts, wantCounts)
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			UPDATE users
			SET name = 'Сохранённое имя'
			WHERE id = 'intern.backend.1'
		`,
	); err != nil {
		t.Fatal(err)
	}

	extra := users.User{
		ID:   "extra",
		Name: "Другой пользователь",
		Role: users.RoleIntern,
		Team: "backend",
	}
	if err := insertUser(ctx, repo.db, extra); err != nil {
		t.Fatal(err)
	}

	if err := users.Seed(ctx, repo.db); err != nil {
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

func TestMentorships(t *testing.T) {
	repo, ctx := testRepository(t)

	for _, user := range []users.User{
		{
			ID:   "a",
			Role: users.RoleIntern,
			Team: "backend",
		},
		{
			ID:   "b",
			Role: users.RoleIntern,
			Team: "backend",
		},
		{
			ID:   "c",
			Role: users.RoleIntern,
			Team: "platform",
		},
		{
			ID:   "mentor",
			Role: users.RoleMentor,
			Team: "backend",
		},
		{
			ID:   "other",
			Role: users.RoleMentor,
			Team: "platform",
		},
	} {
		if err := insertUser(ctx, repo.db, user); err != nil {
			t.Fatal(err)
		}
	}

	for _, link := range []users.Mentorship{
		{
			InternID: "b",
			MentorID: "mentor",
		},
		{
			InternID: "a",
			MentorID: "mentor",
		},
		{
			InternID: "c",
			MentorID: "other",
		},
	} {
		if err := insertMentorship(ctx, repo.db, link); err != nil {
			t.Fatal(err)
		}
	}

	interns, err := repo.ListInterns(ctx, "mentor")
	if err != nil || len(interns) != 2 || interns[0].ID != "a" || interns[1].ID != "b" {
		t.Fatalf("interns: %+v %v", interns, err)
	}

	for _, id := range []string{"missing", "a"} {
		empty, err := repo.ListInterns(ctx, id)
		if err != nil || empty == nil || len(empty) != 0 {
			t.Fatalf("empty: %+v %v", empty, err)
		}
	}

	for _, link := range []users.Mentorship{
		{
			InternID: "a",
			MentorID: "mentor",
		}, // Duplicate.
		{
			InternID: "a",
			MentorID: "other",
		}, // An intern cannot have two mentors.
		{
			InternID: "missing",
			MentorID: "mentor",
		},
		{
			InternID: "a",
			MentorID: "missing",
		},
	} {
		if err := insertMentorship(ctx, repo.db, link); err == nil {
			t.Fatalf("invalid link accepted: %+v", link)
		}
	}
	// Referential integrity also applies to direct writes.
	if _, err := repo.db.ExecContext(
		ctx,
		`
			INSERT INTO mentorships (intern_id,mentor_id)
			VALUES ('missing','mentor')
		`,
	); err == nil {
		t.Fatal("orphan intern accepted")
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			INSERT INTO mentorships (intern_id,mentor_id)
			VALUES ('other','missing')
		`,
	); err == nil {
		t.Fatal("orphan mentor accepted")
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			DELETE
			FROM users
			WHERE id='a'
		`,
	); err != nil {
		t.Fatal(err)
	}

	interns, err = repo.ListInterns(ctx, "mentor")
	if err != nil || len(interns) != 1 || interns[0].ID != "b" {
		t.Fatalf("delete intern: %+v %v", interns, err)
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			DELETE
			FROM users
			WHERE id='mentor'
		`,
	); err != nil {
		t.Fatal(err)
	}

	interns, err = repo.ListInterns(ctx, "mentor")
	if err != nil || len(interns) != 0 {
		t.Fatalf("delete mentor: %+v %v", interns, err)
	}

	if _, err := repo.Get(ctx, "b"); err != nil {
		t.Fatal("deleting mentor deleted intern", err)
	}
}

func TestSeedMentorships(t *testing.T) {
	repo, ctx := testRepository(t)
	// A failure when inserting a relationship must roll back seeded users as well.
	if _, err := repo.db.ExecContext(
		ctx,
		`
			ALTER TABLE mentorships ADD CHECK (intern_id <> 'intern.platform.1')
		`,
	); err != nil {
		t.Fatal(err)
	}

	if err := users.Seed(ctx, repo.db); err == nil {
		t.Fatal("seed accepted rejected mentorship")
	}

	for _, query := range []string{`
		SELECT count(*)
		FROM users
	`, `
		SELECT count(*)
		FROM mentorships
	`} {
		var count int
		if err := repo.db.QueryRowContext(ctx, query).Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial seed: %d %v", count, err)
		}
	}

	if _, err := repo.db.ExecContext(
		ctx,
		`
			ALTER TABLE mentorships DROP CONSTRAINT mentorships_intern_id_check
		`,
	); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := users.Seed(ctx, repo.db); err != nil {
			t.Fatal(err)
		}
	}

	for _, team := range []string{"backend", "platform"} {
		interns, err := repo.ListInterns(ctx, "mentor."+team)
		if err != nil || len(interns) != 2 || interns[0].ID != "intern."+team+".1" || interns[1].ID != "intern."+team+".2" {
			t.Fatalf("seed links: %+v %v", interns, err)
		}
	}
	// Existing assignments are not replaced by the seed.
	if _, err := repo.db.ExecContext(
		ctx,
		`
			UPDATE mentorships
			SET mentor_id='mentor.platform'
			WHERE intern_id='intern.backend.1'
		`,
	); err != nil {
		t.Fatal(err)
	}

	if err := users.Seed(ctx, repo.db); err != nil {
		t.Fatal(err)
	}

	interns, err := repo.ListInterns(ctx, "mentor.platform")
	if err != nil || len(interns) != 3 || interns[0].ID != "intern.backend.1" {
		t.Fatalf("seed replaced assignment: %+v %v", interns, err)
	}
}

func TestGetMentor(t *testing.T) {
	r, ctx := testRepository(t)
	if err := users.Seed(ctx, r.db); err != nil {
		t.Fatal(err)
	}

	want, err := r.Get(ctx, "mentor.backend")
	if err != nil {
		t.Fatal(err)
	}

	got, err := r.GetMentor(ctx, "intern.backend.1")
	if err != nil || got != want {
		t.Fatalf("mentor: %+v %v", got, err)
	}

	for _, id := range []string{"missing", "mentor.backend"} {
		if _, err := r.GetMentor(ctx, id); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("%s: %v", id, err)
		}
	}

	if _, err := r.db.ExecContext(
		ctx,
		`
			UPDATE mentorships
			SET mentor_id='mentor.platform'
			WHERE intern_id='intern.backend.1'
		`,
	); err != nil {
		t.Fatal(err)
	}

	got, err = r.GetMentor(ctx, "intern.backend.1")
	if err != nil || got.ID != "mentor.platform" || got.Name == "" {
		t.Fatalf("reassigned: %+v %v", got, err)
	}

	if _, err := r.db.ExecContext(
		ctx,
		`
			DELETE
			FROM mentorships
			WHERE intern_id='intern.backend.1'
		`,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := r.GetMentor(ctx, "intern.backend.1"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unassigned: %v", err)
	}
}

func insertUser(ctx context.Context, db *sql.DB, user users.User) error {
	_, err := db.ExecContext(
		ctx,
		`
			INSERT INTO users (id, name, role, team)
			VALUES ($1, $2, $3, $4)
		`,
		user.ID,
		user.Name,
		user.Role,
		user.Team,
	)

	return err
}

func insertMentorship(ctx context.Context, db *sql.DB, link users.Mentorship) error {
	_, err := db.ExecContext(
		ctx,
		`
			INSERT INTO mentorships (intern_id, mentor_id)
			VALUES ($1, $2)
		`,
		link.InternID,
		link.MentorID,
	)

	return err
}
