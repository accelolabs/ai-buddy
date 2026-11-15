package buddy

import (
	"backend-ai/internal/platform/cleanup"
	"context"
	"encoding/json"
	"reflect"
)

func (p *Store) Draft(ctx context.Context, a Action) (Action, error) {
	a.ID = ID()

	return scanAction(
		p.DB.QueryRowContext(
			ctx,
			"INSERT INTO actions(id,run_id,owner_id,kind,status,fields) SELECT $1,id,owner_id,$4,'draft',$5 FROM runs WHERE id=$2 AND owner_id=$3 AND status IN ('queued','running') RETURNING "+actionCols,
			a.ID,
			a.RunID,
			a.Owner,
			a.Kind,
			[]byte(a.Fields),
		),
	)
}

func (p *Store) Claim(
	ctx context.Context,
	owner, id string,
	fields json.RawMessage,
) (a Action, execute bool, err error) {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return a, false, err
	}
	defer cleanup.Rollback(tx)

	a, err = scanAction(
		tx.QueryRowContext(
			ctx,
			"SELECT "+actionCols+" FROM actions WHERE id=$1 AND owner_id=$2 FOR UPDATE",
			id,
			owner,
		),
	)
	if err != nil {
		return a, false, err
	}

	if a.Status == ActionCancelled || a.Status == ActionExecuting {
		return a, false, ErrConflict
	}

	if a.Status != ActionDraft {
		if err := frozenFields(a.Fields, fields); err != nil {
			return a, false, err
		}

		if a.Status == ActionDone {
			return a, false, nil
		}
	} else if len(fields) > 0 && string(fields) != "null" {
		a.Fields = fields
	}

	if _, err = parseFields(a.Kind, a.Fields); err != nil {
		return a, false, err
	}

	a.Status = ActionExecuting
	execute = true

	_, err = tx.ExecContext(
		ctx,
		"UPDATE actions SET status=$1,fields=$2,error='' WHERE id=$3",
		a.Status,
		[]byte(a.Fields),
		id,
	)
	if err != nil {
		return a, false, err
	}

	err = tx.Commit()

	return a, execute, err
}

func (p *Store) ActionResult(
	ctx context.Context,
	id string,
	result json.RawMessage,
	opErr error,
) (Action, error) {
	status, failure := ActionDone, ""

	if opErr != nil {
		status = ActionError
		failure = opErr.Error()
	}

	if len(result) == 0 {
		result = json.RawMessage("null")
	}

	return scanAction(
		p.DB.QueryRowContext(
			ctx,
			"UPDATE actions SET status=$1,result=$2,error=$3 WHERE id=$4 AND status='executing' RETURNING "+actionCols,
			status,
			[]byte(result),
			failure,
			id,
		),
	)
}

// Cancel only transitions a draft; completed or in-flight writes cannot be cancelled.
func (p *Store) Cancel(ctx context.Context, owner, id string) (a Action, err error) {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return a, err
	}
	defer cleanup.Rollback(tx)

	a, err = scanAction(
		tx.QueryRowContext(
			ctx,
			"SELECT "+actionCols+" FROM actions WHERE id=$1 AND owner_id=$2 FOR UPDATE",
			id,
			owner,
		),
	)
	if err != nil {
		return a, err
	}

	if a.Status == ActionCancelled {
		return a, nil
	}

	if a.Status != ActionDraft {
		return a, ErrConflict
	}

	a.Status = ActionCancelled

	_, err = tx.ExecContext(
		ctx,
		"UPDATE actions SET status=$1,fields=$2,error='' WHERE id=$3",
		a.Status,
		[]byte(a.Fields),
		id,
	)
	if err != nil {
		return a, err
	}

	return a, tx.Commit()
}

func frozenFields(saved, fields json.RawMessage) error {
	var old, requested any
	if err := json.Unmarshal(saved, &old); err != nil {
		return err
	}

	if len(fields) > 0 && string(fields) != "null" {
		if json.Unmarshal(fields, &requested) != nil || !reflect.DeepEqual(old, requested) {
			return ErrConflict
		}
	}

	return nil
}
