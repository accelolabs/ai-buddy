package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ai-buddy/internal/wiki"
	"ai-buddy/internal/wiki/indexer"
)

var errDBFailure = errors.New("database failed")

func mockRepo(t *testing.T) (*Repository, sqlmock.Sqlmock) {
	t.Helper()
	db, m, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		// sqlmock cleanup after assertions; Close is not part of this scenario.
		_ = db.Close()
	})
	return &Repository{db}, m
}

func state(m sqlmock.Sqlmock, fp string) {
	m.ExpectQuery("SELECT fingerprint").WillReturnRows(sqlmock.NewRows([]string{"fingerprint"}).AddRow(fp))
}

func TestRepositoryReads(t *testing.T) {
	ctx := context.Background()
	t.Run("fingerprint", func(t *testing.T) {
		r, m := mockRepo(t)
		indexLock(m)
		session, err := r.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		state(m, "fp")
		if got, err := session.Fingerprint(ctx); err != nil || got != "fp" {
			t.Fatal(got, err)
		}
		m.ExpectQuery("SELECT fingerprint").WillReturnError(sql.ErrNoRows)
		if got, err := session.Fingerprint(ctx); err != nil || got != "" {
			t.Fatal(got, err)
		}
		indexUnlock(m)
		session.Close()
	})
	t.Run("list", func(t *testing.T) {
		r, m := mockRepo(t)
		m.ExpectQuery("SELECT id,path,title FROM documents WHERE team IN").WithArgs("backend").WillReturnRows(sqlmock.NewRows([]string{"id", "path", "title"}).AddRow("a", "common/a.md", "A"))
		d, err := r.List(ctx, "backend")
		if err != nil || len(d) != 1 {
			t.Fatal(d, err)
		}
	})
	for _, mode := range []string{"query", "scan", "rows"} {
		t.Run("list-"+mode, func(t *testing.T) {
			r, m := mockRepo(t)
			q := m.ExpectQuery("SELECT id,path,title")
			switch mode {
			case "query":
				q.WillReturnError(errDBFailure)
			case "scan":
				q.WillReturnRows(sqlmock.NewRows([]string{"id", "path", "title"}).AddRow(nil, "p", "t"))
			case "rows":
				q.WillReturnRows(sqlmock.NewRows([]string{"id", "path", "title"}).AddRow("a", "p", "t").RowError(0, errDBFailure))
			}
			if _, err := r.List(ctx, "backend"); err == nil {
				t.Fatal("expected error")
			}
		})
	}
	for _, mode := range []string{"ok", "missing", "doc", "scan"} {
		t.Run("get-"+mode, func(t *testing.T) {
			r, m := mockRepo(t)
			query := m.ExpectQuery("SELECT id,path,title,markdown").WithArgs("a", "backend")
			switch mode {
			case "missing":
				query.WillReturnError(sql.ErrNoRows)
			case "doc":
				query.WillReturnError(errDBFailure)
			case "scan":
				query.WillReturnRows(sqlmock.NewRows([]string{"id", "path", "title", "markdown"}).AddRow(nil, "common/a.md", "Title", "# Title"))
			default:
				query.WillReturnRows(sqlmock.NewRows([]string{"id", "path", "title", "markdown"}).AddRow("a", "common/a.md", "Title", "# Title"))
			}
			document, err := r.Get(ctx, "backend", "a")
			if mode == "ok" {
				if err != nil || document.Markdown != "# Title" {
					t.Fatal(document, err)
				}
			} else if err == nil {
				t.Fatal("expected error")
			}
			if mode == "missing" && !errors.Is(err, wiki.ErrNotFound) {
				t.Fatal(err)
			}
		})
	}
}

func TestRepositorySearch(t *testing.T) {
	for _, mode := range []string{"ok", "empty", "begin", "state", "mismatch", "query", "scan", "rows", "commit"} {
		t.Run(mode, func(t *testing.T) {
			r, m := mockRepo(t)
			if mode == "begin" {
				m.ExpectBegin().WillReturnError(errDBFailure)
			} else {
				m.ExpectBegin()
				switch mode {
				case "state":
					m.ExpectQuery("SELECT fingerprint").WillReturnError(errDBFailure)
					m.ExpectRollback()
				case "mismatch":
					state(m, "different")
					m.ExpectRollback()
				default:
					state(m, "fp")
					q := m.ExpectQuery("WITH allowed AS MATERIALIZED").WithArgs("backend", "текст", vector(embedding()), 0.35)
					if mode == "query" {
						q.WillReturnError(errDBFailure)
						m.ExpectRollback()
					} else {
						rows := sqlmock.NewRows([]string{"id", "title", "section", "anchor", "text"})
						if mode != "empty" {
							if mode == "scan" {
								rows.AddRow(nil, "Title", "Section", "anchor", "text")
							} else {
								rows.AddRow("a", "Title", "Section", "anchor", "text")
							}
						}
						if mode == "rows" {
							rows.RowError(0, errDBFailure)
						}
						q.WillReturnRows(rows)
						switch mode {
						case "rows", "scan":
							m.ExpectRollback()
						case "commit":
							m.ExpectCommit().WillReturnError(errDBFailure)
						default:
							m.ExpectCommit()
						}
					}
				}
			}
			got, err := r.Search(context.Background(), "backend", "текст", embedding(), 0.35, "fp")
			if mode == "ok" {
				if err != nil || len(got) != 1 {
					t.Fatal(got, err)
				}
			} else if mode == "empty" {
				if err != nil || got == nil || len(got) != 0 {
					t.Fatal(got, err)
				}
			} else if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func indexLock(m sqlmock.Sqlmock) {
	m.ExpectQuery("SELECT pg_try_advisory_lock").WithArgs(lockID).WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(true))
}

func indexUnlock(m sqlmock.Sqlmock) {
	m.ExpectQuery("SELECT pg_advisory_unlock").WithArgs(lockID).WillReturnRows(sqlmock.NewRows([]string{"unlocked"}).AddRow(true))
}

func TestIndexLifecycle(t *testing.T) {
	root := corpus(t, map[string]string{"backend/a.md": "# A\n\nтекст"})
	d := indexer.Parse("backend/a.md", "# A\n\nтекст")
	for _, mode := range []string{"add", "change", "skip", "full", "delete", "embed", "fingerprint", "mismatch", "read", "oldquery", "oldscan", "oldrows", "begin", "deletefail", "insertfail", "sectionfail", "chunkfail", "statefail", "commit"} {
		t.Run(mode, func(t *testing.T) {
			r, m := mockRepo(t)
			e := &fakeEmbedder{fp: "fp"}
			indexLock(m)
			if mode == "fingerprint" {
				e.err = wiki.ErrUnavailable
				indexUnlock(m)
				if _, err := indexer.Run(context.Background(), r, root, e, false); err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if mode == "mismatch" {
				state(m, "other")
			} else {
				state(m, "fp")
			}
			path := root
			if mode == "read" {
				path = root + "/missing"
			}
			if mode != "read" && mode != "mismatch" {
				old := sqlmock.NewRows([]string{"id", "hash"})
				switch mode {
				case "change", "full", "skip":
					hash := d.Hash
					if mode == "change" {
						hash = "old"
					}
					old.AddRow(d.ID, hash)
				case "delete", "deletefail":
					old.AddRow("gone", "hash")
				case "oldscan":
					old.AddRow(nil, "hash")
				case "oldrows":
					old.AddRow("id", "hash").RowError(0, errDBFailure)
				}
				if mode == "oldquery" {
					m.ExpectQuery("SELECT id,hash").WillReturnError(errDBFailure)
				} else {
					m.ExpectQuery("SELECT id,hash").WillReturnRows(old)
				}
				if mode == "embed" {
					e.hook = func() { e.err = wiki.ErrUnavailable }
				}
				if mode != "embed" && mode != "oldquery" && mode != "oldscan" && mode != "oldrows" {
					if mode == "begin" {
						m.ExpectBegin().WillReturnError(errDBFailure)
					} else {
						m.ExpectBegin()
						failed := false
						del := func(id string) {
							q := m.ExpectExec("DELETE FROM documents").WithArgs(id)
							if mode == "deletefail" {
								q.WillReturnError(errDBFailure)
								failed = true
							} else {
								q.WillReturnResult(sqlmock.NewResult(0, 1))
							}
						}
						if mode == "delete" || mode == "deletefail" {
							del("gone")
						}
						if !failed && mode != "skip" {
							del(d.ID)
							q := m.ExpectExec("INSERT INTO documents").WithArgs(d.ID, d.Path, "backend", d.Title, d.Markdown, d.Hash)
							if mode == "insertfail" {
								q.WillReturnError(errDBFailure)
								failed = true
							} else {
								q.WillReturnResult(sqlmock.NewResult(0, 1))
							}
							if !failed {
								q = m.ExpectExec("INSERT INTO sections")
								if mode == "sectionfail" {
									q.WillReturnError(errDBFailure)
									failed = true
								} else {
									q.WillReturnResult(sqlmock.NewResult(0, 1))
								}
							}
							if !failed {
								q = m.ExpectExec("INSERT INTO chunks")
								if mode == "chunkfail" {
									q.WillReturnError(errDBFailure)
									failed = true
								} else {
									q.WillReturnResult(sqlmock.NewResult(0, 1))
								}
							}
						}
						if !failed {
							q := m.ExpectExec("INSERT INTO wiki_state").WithArgs("fp")
							if mode == "statefail" {
								q.WillReturnError(errDBFailure)
								failed = true
							} else {
								q.WillReturnResult(sqlmock.NewResult(0, 1))
							}
						}
						if failed {
							m.ExpectRollback()
						} else if mode == "commit" {
							m.ExpectCommit().WillReturnError(errDBFailure)
						} else {
							m.ExpectCommit()
						}
					}
				}
			}
			indexUnlock(m)
			stats, err := indexer.Run(context.Background(), r, path, e, mode == "full")
			switch mode {
			case "add", "change", "skip", "full", "delete":
				if err != nil {
					t.Fatal(err)
				}
				switch mode {
				case "add":
					if stats.Added != 1 {
						t.Fatal(stats)
					}
				case "change", "full":
					if stats.Changed != 1 {
						t.Fatal(stats)
					}
				case "skip":
					if stats.Skipped != 1 || e.calls != 0 {
						t.Fatal(stats, e.calls)
					}
				case "delete":
					if stats.Deleted != 1 {
						t.Fatal(stats)
					}
				}
			default:
				if err == nil {
					t.Fatal("expected error")
				}
			}
		})
	}
}
