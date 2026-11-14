package postgres

import (
	"database/sql"
	"errors"

	"ai-buddy/internal/buddy"
)

type Repository struct{ DB *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db} }
func missing(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return buddy.ErrNotFound
	}
	return err
}

const runCols = "id,chat_id,owner_id,message_id,status,error,created_at"

type scanner interface{ Scan(...any) error }

func scanRun(row scanner) (r buddy.Run, err error) {
	err = row.Scan(&r.ID, &r.ChatID, &r.Owner, &r.MessageID, &r.Status, &r.Error, &r.CreatedAt)
	return r, missing(err)
}
func scanAction(row scanner) (a buddy.Action, err error) {
	err = row.Scan(&a.ID, &a.RunID, &a.Owner, &a.Kind, &a.Status, &a.Fields, &a.Result, &a.Error)
	return a, missing(err)
}

const actionCols = "id,run_id,owner_id,kind,status,fields,result,error"
