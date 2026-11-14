package tracker

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

var pcols = []string{"authentik_id", "name", "username", "email", "role", "team", "start_date", "goals", "expectations"}
var tcols = []string{"id", "intern_id", "title", "description", "deadline", "created_at", "completed_at"}

func profileRows() *sqlmock.Rows {
	return sqlmock.NewRows(pcols).AddRow("i", "Name", "intern", "i@test", "intern", "backend", "2026-11-01", "Goals", "Expectations")
}
func taskRows() *sqlmock.Rows {
	return sqlmock.NewRows(tcols).AddRow("t", "i", "Task", "Description", "2026-11-14", time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC), nil)
}
func postgres(t *testing.T) (*Postgres, sqlmock.Sqlmock) {
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
	return &Postgres{db}, m
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
	if _, e = p.Intern(ctx, "none"); e != ErrNotFound {
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
func TestRepositoryLists(t *testing.T) {
	for _, kind := range []string{"profiles", "tasks"} {
		t.Run(kind, func(t *testing.T) {
			p, m := postgres(t)
			ctx := context.Background()
			call := func() (int, error) {
				if kind == "profiles" {
					v, e := p.Interns(ctx, "m")
					return len(v), e
				}
				v, e := p.Tasks(ctx, "i")
				return len(v), e
			}
			rows := profileRows()
			if kind == "tasks" {
				rows = taskRows()
			}
			m.ExpectQuery("SELECT").WillReturnRows(rows)
			if n, e := call(); n != 1 || e != nil {
				t.Fatal(n, e)
			}
			m.ExpectQuery("SELECT").WillReturnError(broken)
			if _, e := call(); e != broken {
				t.Fatal(e)
			}
			m.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"bad"}).AddRow("bad"))
			if _, e := call(); e == nil {
				t.Fatal("scan failure")
			}
			cols := pcols
			if kind == "tasks" {
				cols = tcols
			}
			m.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(cols).AddRow(make([]driver.Value, len(cols))...).RowError(0, broken))
			if _, e := call(); e != broken {
				t.Fatal(e)
			}
		})
	}
}
func TestTaskRepository(t *testing.T) {
	p, m := postgres(t)
	ctx := context.Background()
	in := Input{"Task", "Description", "2026-11-14"}
	m.ExpectQuery("INSERT INTO tasks").WithArgs(sqlmock.AnyArg(), "i", in.Title, in.Description, in.Deadline, "k").WillReturnRows(taskRows())
	v, created, e := p.Create(ctx, "i", "k", in)
	if e != nil || !created || v.ID != "t" {
		t.Fatal(v, created, e)
	}
	m.ExpectQuery("INSERT INTO tasks").WillReturnError(broken)
	if _, _, e = p.Create(ctx, "i", "k", in); e != broken {
		t.Fatal(e)
	}
	for _, mode := range []string{"retry", "conflict", "error"} {
		m.ExpectQuery("INSERT INTO tasks").WillReturnError(sql.ErrNoRows)
		q := m.ExpectQuery("SELECT .* FROM tasks WHERE intern_id=\\$1 AND idempotency_key=\\$2").WithArgs("i", "k")
		if mode == "error" {
			q.WillReturnError(broken)
		} else {
			q.WillReturnRows(taskRows())
		}
		input := in
		if mode == "conflict" {
			input.Title = "Other"
		}
		v, created, e = p.Create(ctx, "i", "k", input)
		switch mode {
		case "retry":
			if e != nil || created || v.ID != "t" {
				t.Fatal(v, created, e)
			}
		case "conflict":
			if e != ErrConflict {
				t.Fatal(e)
			}
		case "error":
			if e != broken {
				t.Fatal(e)
			}
		}
	}
	m.ExpectQuery("UPDATE tasks SET completed_at=COALESCE").WithArgs("i", "t").WillReturnRows(taskRows())
	if v, e = p.Complete(ctx, "i", "t"); e != nil || v.ID != "t" {
		t.Fatal(v, e)
	}
	m.ExpectQuery("UPDATE tasks").WillReturnError(sql.ErrNoRows)
	if _, e = p.Complete(ctx, "i", "other"); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}
	m.ExpectQuery("UPDATE tasks").WillReturnError(broken)
	if _, e = p.Complete(ctx, "i", "t"); e != broken {
		t.Fatal(e)
	}
}
