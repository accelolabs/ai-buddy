package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"ai-buddy/internal/tracker/apperror"
	"ai-buddy/internal/tracker/tasks"
)

var errBroken = errors.New("database failed")

var tcols = []string{
	"id",
	"intern_id",
	"title",
	"description",
	"deadline",
	"created_at",
	"completed_at",
}

func taskRows() *sqlmock.Rows {
	return sqlmock.NewRows(tcols).AddRow(
		"t",
		"i",
		"Task",
		"Description",
		"2026-11-14",
		time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC),
		nil,
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

		// sqlmock cleanup after assertions; Close is not part of this scenario.
		_ = db.Close()
	})
	return New(db), m
}

func TestTaskRepository(t *testing.T) {
	p, m := postgres(t)
	ctx := context.Background()
	in := tasks.CreateInput{
		Title:       "Task",
		Description: "Description",
		Deadline:    "2026-11-14",
	}
	m.ExpectQuery("INSERT INTO tasks").WithArgs(sqlmock.AnyArg(), "i", in.Title, in.Description, in.Deadline, "k").WillReturnRows(taskRows())
	v, created, e := p.Create(ctx, "i", "k", in)
	if e != nil || !created || v.ID != "t" {
		t.Fatal(v, created, e)
	}

	m.ExpectQuery("INSERT INTO tasks").WillReturnError(errBroken)
	if _, _, e = p.Create(ctx, "i", "k", in); e != errBroken {
		t.Fatal(e)
	}

	for _, mode := range []string{"retry", "conflict", "error"} {
		m.ExpectQuery("INSERT INTO tasks").WillReturnError(sql.ErrNoRows)
		q := m.ExpectQuery("SELECT .* FROM tasks WHERE intern_id=\\$1 AND idempotency_key=\\$2").WithArgs("i", "k")
		if mode == "error" {
			q.WillReturnError(errBroken)
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
			if e != apperror.ErrConflict {
				t.Fatal(e)
			}
		case "error":
			if e != errBroken {
				t.Fatal(e)
			}
		}
	}

	m.ExpectQuery("UPDATE tasks SET completed_at=COALESCE").WithArgs("i", "t").WillReturnRows(taskRows())
	if v, e = p.Complete(ctx, "i", "t"); e != nil || v.ID != "t" {
		t.Fatal(v, e)
	}

	m.ExpectQuery("UPDATE tasks").WillReturnError(sql.ErrNoRows)
	if _, e = p.Complete(ctx, "i", "other"); !errors.Is(e, apperror.ErrNotFound) {
		t.Fatal(e)
	}

	m.ExpectQuery("UPDATE tasks").WillReturnError(errBroken)
	if _, e = p.Complete(ctx, "i", "t"); e != errBroken {
		t.Fatal(e)
	}
}

func TestList(t *testing.T) {
	repo, m := postgres(t)
	ctx := context.Background()
	m.ExpectQuery("SELECT").WillReturnRows(taskRows())
	list, err := repo.Tasks(ctx, "i")
	if err != nil || len(list) != 1 {
		t.Fatal(list, err)
	}

	m.ExpectQuery("SELECT").WillReturnError(errBroken)
	if _, err = repo.Tasks(ctx, "i"); err != errBroken {
		t.Fatal(err)
	}

	m.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"bad"}).AddRow("bad"))
	if _, err = repo.Tasks(ctx, "i"); err == nil {
		t.Fatal("scan failure")
	}

	m.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(tcols).AddRow(make([]driver.Value, len(tcols))...).RowError(0, errBroken))
	if _, err = repo.Tasks(ctx, "i"); err != errBroken {
		t.Fatal(err)
	}
}
