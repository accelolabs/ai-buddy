package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"ai-buddy/backend-tracker/internal/apperror"
	"github.com/DATA-DOG/go-sqlmock"
)

var broken = errors.New("database failed")

var pcols = []string{
	"authentik_id",
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

func postgres(t *testing.T) (*Repository, sqlmock.Sqlmock) {
	t.Helper()
	db, m, e := sqlmock.New()
	if e != nil {
		t.Fatal(e)
	}

	t.Cleanup(func() {
		if e := m.ExpectationsWereMet(); e != nil {
			t.Error(e)
		}

		db.Close()
	})
	return New(db), m
}

func TestProfileRepository(t *testing.T) {
	p, m := postgres(t)
	ctx := context.Background()
	m.ExpectQuery("SELECT .* FROM profiles WHERE authentik_id=\\$1").WithArgs("i").WillReturnRows(profileRows())
	v, e := p.Intern(ctx, "i")
	if e != nil || v.ID != "i" || *v.StartDate != "2026-11-01" {
		t.Fatal(v, e)
	}

	m.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)
	if _, e = p.Intern(ctx, "none"); e != apperror.ErrNotFound {
		t.Fatal(e)
	}

	m.ExpectQuery("SELECT").WillReturnError(broken)
	if _, e = p.Intern(ctx, "i"); e != broken {
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

	m.ExpectQuery("SELECT").WillReturnError(broken)
	if _, e = p.Mentor(ctx, "i"); e != broken {
		t.Fatal(e)
	}

	m.ExpectQuery("SELECT EXISTS").WithArgs("m", "i").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	ok, e := p.Assigned(ctx, "m", "i")
	if !ok || e != nil {
		t.Fatal(ok, e)
	}

	m.ExpectQuery("SELECT EXISTS").WillReturnError(broken)
	if _, e = p.Assigned(ctx, "m", "i"); e != broken {
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

	m.ExpectQuery("SELECT").WillReturnError(broken)
	if _, err = repo.Interns(ctx, "i"); err != broken {
		t.Fatal(err)
	}

	m.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"bad"}).AddRow("bad"))
	if _, err = repo.Interns(ctx, "i"); err == nil {
		t.Fatal("scan failure")
	}

	m.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(pcols).AddRow(make([]driver.Value, len(pcols))...).RowError(0, broken))
	if _, err = repo.Interns(ctx, "i"); err != broken {
		t.Fatal(err)
	}
}
