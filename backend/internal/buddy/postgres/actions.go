package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"reflect"

	"ai-buddy/internal/buddy"
)

func (p *Repository) Draft(ctx context.Context, a buddy.Action) (buddy.Action, error) {
	a.ID = buddy.ID()
	a.Status = buddy.ActionDraft
	return scanAction(p.DB.QueryRowContext(ctx, "INSERT INTO actions(id,run_id,owner_id,kind,status,fields) SELECT $1,id,owner_id,$4,'draft',$5 FROM runs WHERE id=$2 AND owner_id=$3 AND status IN ('queued','running') RETURNING "+actionCols, a.ID, a.RunID, a.Owner, a.Kind, []byte(a.Fields)))
}
func (p *Repository) Claim(ctx context.Context, owner, id string, fields json.RawMessage) (a buddy.Action, execute bool, err error) {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return a, false, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Warn("resource cleanup failed", "error", err)
		}
	}()
	a, err = scanAction(tx.QueryRowContext(ctx, "SELECT "+actionCols+" FROM actions WHERE id=$1 AND owner_id=$2 FOR UPDATE", id, owner))
	if err != nil {
		return a, false, err
	}

	if a.Status == buddy.ActionCancelled || a.Status == buddy.ActionExecuting {
		return a, false, buddy.ErrConflict
	}
	if a.Status != buddy.ActionDraft {
		var old, new any
		if err = json.Unmarshal(a.Fields, &old); err != nil {
			return a, false, err
		}
		if len(fields) > 0 && string(fields) != "null" {
			if json.Unmarshal(fields, &new) != nil || !reflect.DeepEqual(old, new) {
				return a, false, buddy.ErrConflict
			}
		}
		if a.Status == buddy.ActionDone {
			return a, false, nil
		}
	} else if len(fields) > 0 && string(fields) != "null" {
		a.Fields = fields
	}
	if err = buddy.ValidateFields(a.Kind, a.Fields); err != nil {
		return a, false, err
	}
	a.Status = buddy.ActionExecuting
	execute = true
	_, err = tx.ExecContext(ctx, "UPDATE actions SET status=$1,fields=$2,error='' WHERE id=$3", a.Status, []byte(a.Fields), id)
	if err != nil {
		return a, false, err
	}
	err = tx.Commit()
	return
}
func (p *Repository) ActionResult(ctx context.Context, id string, result json.RawMessage, opErr error) (buddy.Action, error) {
	status, failure := buddy.ActionDone, ""
	if opErr != nil {
		status = buddy.ActionError
		failure = opErr.Error()
	}
	if len(result) == 0 {
		result = json.RawMessage("null")
	}
	return scanAction(p.DB.QueryRowContext(ctx, "UPDATE actions SET status=$1,result=$2,error=$3 WHERE id=$4 AND status='executing' RETURNING "+actionCols, status, []byte(result), failure, id))
}

// Cancel only transitions a draft; completed or in-flight writes cannot be cancelled.
func (p *Repository) Cancel(ctx context.Context, owner, id string) (a buddy.Action, err error) {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return a, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Warn("resource cleanup failed", "error", err)
		}
	}()
	a, err = scanAction(tx.QueryRowContext(ctx, "SELECT "+actionCols+" FROM actions WHERE id=$1 AND owner_id=$2 FOR UPDATE", id, owner))
	if err != nil {
		return a, err
	}
	if a.Status == buddy.ActionCancelled {
		return a, nil
	}
	if a.Status != buddy.ActionDraft {
		return a, buddy.ErrConflict
	}
	a.Status = buddy.ActionCancelled
	_, err = tx.ExecContext(ctx, "UPDATE actions SET status=$1,fields=$2,error='' WHERE id=$3", a.Status, []byte(a.Fields), id)
	if err != nil {
		return a, err
	}
	return a, tx.Commit()
}
