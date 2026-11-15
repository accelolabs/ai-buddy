package users

import (
	"backend-ai/internal/platform/cleanup"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

var errBroken = errors.New("database failed")

var pcols = []string{
	"id",
	"name",
	"username",
	"email",
	"role",
	"team",
	"start_date",
	"goals",
	"expectations",
}

func profileRows() *sqlmock.Rows {
	return sqlmock.NewRows(pcols).AddRow(
		"i",
		"Name",
		"intern",
		"i@test",
		"intern",
		"backend",
		"2026-11-01",
		"Goals",
		"Expectations",
	)
}

func postgres(t *testing.T) (*Store, sqlmock.Sqlmock) {
	t.Helper()

	db, m, e := sqlmock.New()
	if e != nil {
		t.Fatal(e)
	}

	t.Cleanup(func() {
		if e := m.ExpectationsWereMet(); e != nil {
			t.Error(e)
		}

		// sqlmock cleanup after assertions; Close is not part of this scenario.
		cleanup.Close(db)
	})

	return &Store{DB: db}, m
}

func TestProfileRepository(t *testing.T) {
	p, m := postgres(t)
	ctx := context.Background()

	m.ExpectQuery("SELECT .* FROM profiles WHERE id=\\$1").
		WithArgs("i").
		WillReturnRows(profileRows())

	v, e := p.Get(ctx, "i")
	if e != nil || v.ID != "i" || *v.StartDate != "2026-11-01" {
		t.Fatal(v, e)
	}

	m.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	if _, e = p.Get(ctx, "none"); !errors.Is(e, sql.ErrNoRows) {
		t.Fatal(e)
	}

	m.ExpectQuery("SELECT").WillReturnError(errBroken)

	if _, e = p.Get(ctx, "i"); !errors.Is(e, errBroken) {
		t.Fatal(e)
	}

	m.ExpectQuery("SELECT").WithArgs("i").WillReturnRows(profileRows())

	mentor, e := p.Mentor(ctx, "i")
	if e != nil || mentor == nil {
		t.Fatal(mentor, e)
	}

	m.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	if mentor, e = p.Mentor(ctx, "j"); e != nil || mentor != nil {
		t.Fatal(mentor, e)
	}

	m.ExpectQuery("SELECT").WillReturnError(errBroken)

	if _, e = p.Mentor(ctx, "i"); !errors.Is(e, errBroken) {
		t.Fatal(e)
	}

	m.ExpectQuery("SELECT EXISTS").
		WithArgs("m", "i").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	ok, e := p.Assigned(ctx, "m", "i")
	if !ok || e != nil {
		t.Fatal(ok, e)
	}

	m.ExpectQuery("SELECT EXISTS").WillReturnError(errBroken)

	if _, e = p.Assigned(ctx, "m", "i"); !errors.Is(e, errBroken) {
		t.Fatal(e)
	}
}

func TestList(t *testing.T) {
	repo, m := postgres(t)
	ctx := context.Background()

	m.ExpectQuery("SELECT").WillReturnRows(profileRows())

	list, err := repo.Interns(ctx, "i")
	if err != nil || len(list) != 1 {
		t.Fatal(list, err)
	}

	m.ExpectQuery("SELECT").WillReturnError(errBroken)

	if _, err = repo.Interns(ctx, "i"); !errors.Is(err, errBroken) {
		t.Fatal(err)
	}

	m.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"bad"}).AddRow("bad"))

	if _, err = repo.Interns(ctx, "i"); err == nil {
		t.Fatal("scan failure")
	}

	m.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows(pcols).AddRow(make([]driver.Value, len(pcols))...).RowError(0, errBroken))

	if _, err = repo.Interns(ctx, "i"); !errors.Is(err, errBroken) {
		t.Fatal(err)
	}
}
