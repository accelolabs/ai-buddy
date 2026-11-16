package wiki

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"math"
	"reflect"
	"regexp"
	"testing"

	// Project packages.
	"backend/internal/wiki"

	"github.com/DATA-DOG/go-sqlmock"
)

const listSQL = `
			SELECT id, path, team, title
			FROM documents
			ORDER BY path
		`

const getSQL = `
			SELECT id, path, team, title
			FROM documents
			WHERE id = $1
		`

const deleteDocumentsSQL = `
			DELETE
			FROM documents
			WHERE NOT (path = ANY($1::text[]))
		`

const upsertSQL = `
				INSERT INTO documents (id, path, team, title)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (path) DO UPDATE
				SET team = EXCLUDED.team, title = EXCLUDED.title
				WHERE (documents.team, documents.title) IS DISTINCT FROM (EXCLUDED.team, EXCLUDED.title)
			`

const deleteChunksSQL = `
			DELETE
			FROM chunks
		`

const insertChunkSQL = `
				INSERT INTO chunks (document_id, ordinal, anchor, text, embedding)
				VALUES ($1, $2, $3, $4, $5)
			`

const searchSQL = `
			SELECT document_id, ordinal, anchor, text
			FROM chunks
			WHERE document_id = ANY($1::text[])
				AND embedding IS NOT NULL
			ORDER BY embedding OPERATOR(public.<=>) $2::public.vector, document_id, ordinal
			LIMIT $3
		`

func mockRepository(t *testing.T) (*Repository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(arrayConverter{}))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}

		_ = db.Close()
	})

	return NewRepository(db), mock
}

var errQueryFailure = errors.New("query failure")

func TestReadDocuments(t *testing.T) {
	expected := wiki.Document{ID: "doc", Path: "guide.md", Team: "backend", Title: "Guide"}

	for _, single := range []bool{false, true} {
		for _, mode := range []string{"ok", "empty", "query", "scan", "iterate"} {
			t.Run(fmtName(single, mode), func(t *testing.T) {
				repo, mock := mockRepository(t)
				query := listSQL
				if single {
					query = getSQL
				}

				e := mock.ExpectQuery(regexp.QuoteMeta(query))
				if single {
					e.WithArgs("doc")
				}

				rows := sqlmock.NewRows([]string{"id", "path", "team", "title"})
				switch mode {
				case "ok":
					rows.AddRow("doc", "guide.md", "backend", "Guide")
				case "scan":
					rows.AddRow(nil, "guide.md", "backend", "Guide")
				case "iterate":
					rows.AddRow("doc", "guide.md", "backend", "Guide").RowError(0, errQueryFailure)
				}

				if mode == "query" {
					e.WillReturnError(errQueryFailure)
				} else {
					e.WillReturnRows(rows).RowsWillBeClosed()
				}

				if single {
					got, err := repo.Get(t.Context(), "doc")
					if mode == "ok" {
						if err != nil || got != expected {
							t.Fatalf("%+v %v", got, err)
						}
					} else if err == nil {
						t.Fatal("missing error")
					}

					if mode == "empty" && !errors.Is(err, sql.ErrNoRows) {
						t.Fatal(err)
					}
				} else {
					got, err := repo.List(t.Context())
					switch mode {
					case "ok":
						if err != nil || !reflect.DeepEqual(got, []wiki.Document{expected}) {
							t.Fatalf("%+v %v", got, err)
						}
					case "empty":
						if err != nil || got == nil || len(got) != 0 {
							t.Fatalf("%+v %v", got, err)
						}
					default:
						if err == nil {
							t.Fatal("missing error")
						}
					}
				}
			})
		}
	}
}

func fmtName(single bool, mode string) string {
	if single {
		return "get/" + mode
	}

	return "list/" + mode
}

func TestSearch(t *testing.T) {
	for _, mode := range []string{"ok", "empty", "query", "scan", "iterate", "encode"} {
		t.Run(mode, func(t *testing.T) {
			repo, mock := mockRepository(t)
			vector := []float64{1, 0.5}
			if mode == "encode" {
				vector = []float64{math.NaN()}
			} else {
				e := mock.ExpectQuery(regexp.QuoteMeta(searchSQL)).WithArgs([]string{"doc"}, "[1,0.5]", 3)
				rows := sqlmock.NewRows([]string{"document_id", "ordinal", "anchor", "text"})
				switch mode {
				case "ok":
					rows.AddRow("doc", 2, "section", "Text")
				case "scan":
					rows.AddRow("doc", "bad ordinal", "section", "Text")
				case "iterate":
					rows.AddRow("doc", 2, "section", "Text").RowError(0, errQueryFailure)
				}

				if mode == "query" {
					e.WillReturnError(errQueryFailure)
				} else {
					e.WillReturnRows(rows).RowsWillBeClosed()
				}
			}

			got, err := repo.SearchChunks(t.Context(), vector, []string{"doc"}, 3)
			switch mode {
			case "ok":
				expected := []wiki.Chunk{{DocumentID: "doc", Ordinal: 2, Anchor: "section", Text: "Text"}}
				if err != nil || !reflect.DeepEqual(got, expected) {
					t.Fatalf("%+v %v", got, err)
				}
			case "empty":
				if err != nil || got == nil || len(got) != 0 {
					t.Fatalf("%+v %v", got, err)
				}
			default:
				if err == nil {
					t.Fatal("missing error")
				}
			}
		})
	}
}

func TestSync(t *testing.T) {
	for _, fail := range []string{
		"", "begin", "delete docs", "upsert", "delete chunks", "encode", "insert chunk", "commit",
	} {
		t.Run("failure/"+fail, func(t *testing.T) {
			repo, mock := mockRepository(t)
			docs := []wiki.Document{{ID: "doc", Path: "guide.md", Team: "backend", Title: "Guide"}}
			chunks := []wiki.Chunk{
				{DocumentID: "doc", Ordinal: 2, Anchor: "section", Text: "Text", Embedding: []float64{1, 0.5}},
				{DocumentID: "doc", Ordinal: 3, Text: "No vector"},
			}

			begin := mock.ExpectBegin()
			if fail == "begin" {
				begin.WillReturnError(errQueryFailure)
			} else {
				operations := []struct {
					name, query string
					args        []driver.Value
				}{
					{"delete docs", deleteDocumentsSQL, []driver.Value{[]string{"guide.md"}}},
					{"upsert", upsertSQL, []driver.Value{"doc", "guide.md", "backend", "Guide"}},
					{"delete chunks", deleteChunksSQL, nil},
					{"insert chunk", insertChunkSQL, []driver.Value{"doc", 2, "section", "Text", "[1,0.5]"}},
					{"insert second", insertChunkSQL, []driver.Value{"doc", 3, "", "No vector", nil}},
				}

				for _, op := range operations {
					if fail == "encode" && op.name == "insert chunk" {
						chunks[0].Embedding = []float64{math.Inf(1)}
						break
					}

					e := mock.ExpectExec(regexp.QuoteMeta(op.query)).WithArgs(op.args...)
					if op.name == fail {
						e.WillReturnError(errQueryFailure)
						break
					}

					e.WillReturnResult(sqlmock.NewResult(0, 1))
				}

				if fail == "" || fail == "commit" {
					e := mock.ExpectCommit()
					if fail == "commit" {
						e.WillReturnError(errQueryFailure)
					}
				} else {
					mock.ExpectRollback()
				}
			}

			err := repo.Sync(t.Context(), docs, chunks)
			if (err != nil) != (fail != "") {
				t.Fatalf("%v", err)
			}
		})
	}
}

// pgx accepts PostgreSQL arrays; sqlmock needs a converter to retain their values.
type arrayConverter struct{}

func (arrayConverter) ConvertValue(value any) (driver.Value, error) {
	if array, ok := value.([]string); ok {
		return array, nil
	}

	return driver.DefaultParameterConverter.ConvertValue(value)
}

func TestEmptySync(t *testing.T) {
	repo, mock := mockRepository(t)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(deleteDocumentsSQL)).WithArgs([]string{}).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(deleteChunksSQL)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.Sync(t.Context(), nil, nil); err != nil {
		t.Fatal(err)
	}
}
