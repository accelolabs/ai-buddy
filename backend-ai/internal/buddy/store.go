package buddy

import (
	"backend-ai/internal/platform/database"
	"database/sql"
	"errors"
)

type Store struct{ DB *sql.DB }

func missing(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	return err
}

const runCols = "id,chat_id,owner_id,message_id,status,error,created_at"

func scanRun(row database.RowScanner) (r Run, err error) {
	err = row.Scan(&r.ID, &r.ChatID, &r.Owner, &r.MessageID, &r.Status, &r.Error, &r.CreatedAt)
	return r, missing(err)
}

func scanAction(row database.RowScanner) (a Action, err error) {
	err = row.Scan(
		&a.ID,
		&a.RunID,
		&a.Owner,
		&a.Kind,
		&a.Status,
		&a.Fields,
		&a.Result,
		&a.Error,
	)

	return a, missing(err)
}

const actionCols = "id,run_id,owner_id,kind,status,fields,result,error"
