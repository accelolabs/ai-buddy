package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"log/slog"
	"time"

	"ai-buddy/internal/platform/cleanup"
	"ai-buddy/internal/wiki/indexer"
)

const lockID = 824165097

type session struct {
	conn *sql.Conn
}

func (r *Repository) Acquire(ctx context.Context) (indexer.Session, error) {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	var acquired bool
	err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&acquired)
	if err != nil || !acquired {
		cleanup.Close(conn)
		if err == nil {
			err = errors.New("wiki index is already running")
		}
		return nil, err
	}

	return &session{conn: conn}, nil
}

func (s *session) Close() {
	defer cleanup.Close(s.conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var unlocked bool
	err := s.conn.QueryRowContext(ctx, "SELECT pg_advisory_unlock($1)", lockID).Scan(&unlocked)
	if err != nil || !unlocked {
		// A connection with an unreleased session lock must not return to the pool.
		if err := s.conn.Raw(func(any) error { return driver.ErrBadConn }); err != nil {
			slog.Warn("resource cleanup failed", "error", err)
		}
	}
}

func (s *session) Fingerprint(ctx context.Context) (string, error) {
	var fingerprint string
	err := s.conn.QueryRowContext(ctx, "SELECT fingerprint FROM wiki_state WHERE singleton").Scan(&fingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}

	return fingerprint, err
}

func (s *session) Hashes(ctx context.Context) (map[string]string, error) {
	rows, err := s.conn.QueryContext(ctx, "SELECT id,hash FROM documents")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Warn("resource cleanup failed", "error", err)
		}
	}()

	hashes := map[string]string{}
	for rows.Next() {
		var id, hash string
		if err := rows.Scan(&id, &hash); err != nil {
			return nil, err
		}
		hashes[id] = hash
	}

	return hashes, rows.Err()
}
