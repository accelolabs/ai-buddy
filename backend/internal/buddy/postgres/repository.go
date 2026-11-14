package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"

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

func (p *Repository) CreateChat(ctx context.Context, owner string) (c buddy.Chat, err error) {
	c.ID = buddy.ID()
	c.Owner = owner
	err = p.DB.QueryRowContext(ctx, "INSERT INTO chats(id,owner_id) VALUES($1,$2) RETURNING created_at", c.ID, owner).Scan(&c.CreatedAt)
	return
}
func (p *Repository) Chats(ctx context.Context, owner string) ([]buddy.Chat, error) {
	rows, err := p.DB.QueryContext(ctx, "SELECT id,owner_id,title,created_at FROM chats WHERE owner_id=$1 ORDER BY created_at DESC,id", owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []buddy.Chat{}
	for rows.Next() {
		var c buddy.Chat
		if err = rows.Scan(&c.ID, &c.Owner, &c.Title, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (p *Repository) History(ctx context.Context, owner, id string) (h buddy.History, err error) {
	tx, err := p.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return h, err
	}
	defer tx.Rollback()
	err = tx.QueryRowContext(ctx, "SELECT id,owner_id,title,created_at FROM chats WHERE id=$1 AND owner_id=$2", id, owner).Scan(&h.Chat.ID, &h.Chat.Owner, &h.Chat.Title, &h.Chat.CreatedAt)
	if err != nil {
		return h, missing(err)
	}
	h.Messages = []buddy.Message{}
	h.Runs = []buddy.Run{}
	h.Actions = []buddy.Action{}
	rows, err := tx.QueryContext(ctx, "SELECT id,chat_id,run_id,role,content,created_at FROM messages WHERE chat_id=$1 ORDER BY created_at,role DESC,id", id)
	if err != nil {
		return h, err
	}
	for rows.Next() {
		var m buddy.Message
		if err = rows.Scan(&m.ID, &m.ChatID, &m.RunID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			rows.Close()
			return h, err
		}
		h.Messages = append(h.Messages, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return h, err
	}
	rows, err = tx.QueryContext(ctx, "SELECT "+runCols+" FROM runs WHERE chat_id=$1 ORDER BY created_at,id", id)
	if err != nil {
		return h, err
	}
	for rows.Next() {
		r, e := scanRun(rows)
		if e != nil {
			rows.Close()
			return h, e
		}
		h.Runs = append(h.Runs, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return h, err
	}
	rows, err = tx.QueryContext(ctx, "SELECT "+actionCols+" FROM actions WHERE run_id IN (SELECT id FROM runs WHERE chat_id=$1) ORDER BY id", id)
	if err != nil {
		return h, err
	}
	for rows.Next() {
		a, e := scanAction(rows)
		if e != nil {
			rows.Close()
			return h, e
		}
		h.Actions = append(h.Actions, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return h, err
	}

	h.Cards = []buddy.Card{}
	rows, err = tx.QueryContext(ctx, "SELECT e.run_id,e.kind,e.data FROM events e JOIN runs r ON r.id=e.run_id WHERE r.chat_id=$1 AND e.kind IN ('source','result') ORDER BY r.created_at,e.sequence", id)
	if err != nil {
		return h, err
	}
	for rows.Next() {
		var c buddy.Card
		if err = rows.Scan(&c.RunID, &c.Kind, &c.Data); err != nil {
			rows.Close()
			return h, err
		}
		h.Cards = append(h.Cards, c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return h, err
	}
	err = tx.Commit()
	return
}
func (p *Repository) Start(ctx context.Context, owner, chat, key, text string) (buddy.Run, bool, error) {
	return p.start(ctx, owner, chat, "message:"+key, text, "")
}
func (p *Repository) Retry(ctx context.Context, owner, id string) (buddy.Run, bool, error) {
	r, err := p.Run(ctx, owner, id)
	if err != nil {
		return r, false, err
	}
	if r.Status != "error" {
		return r, false, buddy.ErrConflict
	}
	return p.start(ctx, owner, r.ChatID, "retry:"+id, "", id)
}
func (p *Repository) start(ctx context.Context, owner, chat, key, text, retry string) (r buddy.Run, fresh bool, err error) {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return r, false, err
	}
	defer tx.Rollback()
	var title string
	err = tx.QueryRowContext(ctx, "SELECT title FROM chats WHERE id=$1 AND owner_id=$2 FOR UPDATE", chat, owner).Scan(&title)
	if err != nil {
		return r, false, missing(err)
	}
	r, err = scanRun(tx.QueryRowContext(ctx, "SELECT "+runCols+" FROM runs WHERE chat_id=$1 AND request_key=$2", chat, key))
	if err == nil {
		var old string
		err = tx.QueryRowContext(ctx, "SELECT content FROM messages WHERE id=$1", r.MessageID).Scan(&old)
		if err != nil {
			return r, false, err
		}
		if retry == "" && old != text {
			return r, false, buddy.ErrConflict
		}
		return r, false, nil
	}
	if !errors.Is(err, buddy.ErrNotFound) {
		return r, false, err
	}
	var active bool
	err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM runs WHERE chat_id=$1 AND status IN ('queued','running'))", chat).Scan(&active)
	if err != nil {
		return r, false, err
	}
	if active {
		return r, false, buddy.ErrConflict
	}
	r = buddy.Run{ID: buddy.ID(), ChatID: chat, Owner: owner, Status: "queued"}
	var retryID any
	if retry != "" {
		retryID = retry
		err = tx.QueryRowContext(ctx, "SELECT message_id FROM runs WHERE id=$1 AND status='error'", retry).Scan(&r.MessageID)
	} else {
		r.MessageID = buddy.ID()
		_, err = tx.ExecContext(ctx, "INSERT INTO messages(id,chat_id,run_id,role,content) VALUES($1,$2,$3,'user',$4)", r.MessageID, chat, r.ID, text)
	}
	if err != nil {
		return r, false, err
	}
	if title == "" {
		_, err = tx.ExecContext(ctx, "UPDATE chats SET title=$1 WHERE id=$2", buddy.Title(text), chat)
		if err != nil {
			return r, false, err
		}
	}
	err = tx.QueryRowContext(ctx, "INSERT INTO runs(id,chat_id,owner_id,message_id,status,request_key,retry_of) VALUES($1,$2,$3,$4,'queued',$5,$6) RETURNING created_at", r.ID, chat, owner, r.MessageID, key, retryID).Scan(&r.CreatedAt)
	if err != nil {
		return r, false, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO messages(id,chat_id,run_id,role,content) VALUES($1,$2,$3,'assistant','')", buddy.ID(), chat, r.ID)
	if err != nil {
		return r, false, err
	}
	if err = tx.Commit(); err != nil {
		return r, false, err
	}
	return r, true, nil
}
func (p *Repository) Run(ctx context.Context, owner, id string) (buddy.Run, error) {
	return scanRun(p.DB.QueryRowContext(ctx, "SELECT "+runCols+" FROM runs WHERE id=$1 AND owner_id=$2", id, owner))
}
func appendEvent(ctx context.Context, tx *sql.Tx, id, kind string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO events(run_id,sequence,kind,data) SELECT $1,COALESCE(MAX(sequence),0)+1,$2,$3 FROM events WHERE run_id=$1", id, kind, b)
	return err
}
func (p *Repository) Event(ctx context.Context, id, kind string, data any) error {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	err = tx.QueryRowContext(ctx, "SELECT status FROM runs WHERE id=$1 FOR UPDATE", id).Scan(&status)
	if err != nil {
		return missing(err)
	}
	if status != "queued" && status != "running" {
		return buddy.ErrConflict
	}
	if kind == "text" {
		text, ok := data.(string)
		if !ok {
			return buddy.ErrInvalid
		}
		_, err = tx.ExecContext(ctx, "UPDATE messages SET content=content || $1 WHERE run_id=$2 AND role='assistant'", text, id)
	} else if kind == "stage" {
		_, err = tx.ExecContext(ctx, "UPDATE runs SET status='running' WHERE id=$1", id)
	}
	if err != nil {
		return err
	}
	if err = appendEvent(ctx, tx, id, kind, data); err != nil {
		return err
	}
	return tx.Commit()
}
func (p *Repository) Events(ctx context.Context, owner, id string, after int64) ([]buddy.Event, buddy.Run, error) {
	r, err := p.Run(ctx, owner, id)
	if err != nil {
		return nil, r, err
	}
	rows, err := p.DB.QueryContext(ctx, "SELECT sequence,kind,data FROM events WHERE run_id=$1 AND sequence>$2 ORDER BY sequence", id, after)
	if err != nil {
		return nil, r, err
	}
	defer rows.Close()
	out := []buddy.Event{}
	for rows.Next() {
		var e buddy.Event
		if err = rows.Scan(&e.ID, &e.Kind, &e.Data); err != nil {
			return nil, r, err
		}
		out = append(out, e)
	}
	return out, r, rows.Err()
}
func (p *Repository) Finish(ctx context.Context, id, failure string) error {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	err = tx.QueryRowContext(ctx, "SELECT status FROM runs WHERE id=$1 FOR UPDATE", id).Scan(&status)
	if err != nil {
		return missing(err)
	}
	if status == "done" || status == "error" {
		return nil
	}
	status = "done"
	kind := "done"
	if failure != "" {
		status = "error"
		kind = "error"
	}
	_, err = tx.ExecContext(ctx, "UPDATE runs SET status=$1,error=$2 WHERE id=$3", status, failure, id)
	if err != nil {
		return err
	}
	if err = appendEvent(ctx, tx, id, kind, map[string]string{"status": status, "error": failure}); err != nil {
		return err
	}
	return tx.Commit()
}
func (p *Repository) Recover(ctx context.Context) error {
	rows, err := p.DB.QueryContext(ctx, "SELECT id FROM runs WHERE status IN ('queued','running')")
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err = p.Finish(ctx, id, "service restarted; retry manually"); err != nil {
			return err
		}
	}
	_, err = p.DB.ExecContext(ctx, "UPDATE actions SET status='error',error='service restarted; retry manually' WHERE status='executing'")
	return err
}
func (p *Repository) Draft(ctx context.Context, a buddy.Action) (buddy.Action, error) {
	a.ID = buddy.ID()
	a.Status = "draft"
	return scanAction(p.DB.QueryRowContext(ctx, "INSERT INTO actions(id,run_id,owner_id,kind,status,fields) SELECT $1,id,owner_id,$4,'draft',$5 FROM runs WHERE id=$2 AND owner_id=$3 AND status IN ('queued','running') RETURNING "+actionCols, a.ID, a.RunID, a.Owner, a.Kind, []byte(a.Fields)))
}
func (p *Repository) Claim(ctx context.Context, owner, id string, fields json.RawMessage, cancel bool) (a buddy.Action, execute bool, err error) {
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return a, false, err
	}
	defer tx.Rollback()
	a, err = scanAction(tx.QueryRowContext(ctx, "SELECT "+actionCols+" FROM actions WHERE id=$1 AND owner_id=$2 FOR UPDATE", id, owner))
	if err != nil {
		return a, false, err
	}
	if cancel {
		if a.Status == "cancelled" {
			return a, false, nil
		}
		if a.Status != "draft" {
			return a, false, buddy.ErrConflict
		}
		a.Status = "cancelled"
	} else {
		if a.Status == "cancelled" || a.Status == "executing" {
			return a, false, buddy.ErrConflict
		}
		if a.Status != "draft" {
			var old, new any
			_ = json.Unmarshal(a.Fields, &old)
			if len(fields) > 0 && string(fields) != "null" {
				if json.Unmarshal(fields, &new) != nil || !reflect.DeepEqual(old, new) {
					return a, false, buddy.ErrConflict
				}
			}
			if a.Status == "done" {
				return a, false, nil
			}
		} else if len(fields) > 0 && string(fields) != "null" {
			a.Fields = fields
		}
		if err = buddy.ValidateFields(a.Kind, a.Fields); err != nil {
			return a, false, err
		}
		a.Status = "executing"
		execute = true
	}
	_, err = tx.ExecContext(ctx, "UPDATE actions SET status=$1,fields=$2,error='' WHERE id=$3", a.Status, []byte(a.Fields), id)
	if err != nil {
		return a, false, err
	}
	err = tx.Commit()
	return
}
func (p *Repository) ActionResult(ctx context.Context, id string, result json.RawMessage, opErr error) (buddy.Action, error) {
	status, failure := "done", ""
	if opErr != nil {
		status = "error"
		failure = opErr.Error()
	}
	if len(result) == 0 {
		result = json.RawMessage("null")
	}
	return scanAction(p.DB.QueryRowContext(ctx, "UPDATE actions SET status=$1,result=$2,error=$3 WHERE id=$4 AND status='executing' RETURNING "+actionCols, status, []byte(result), failure, id))
}
