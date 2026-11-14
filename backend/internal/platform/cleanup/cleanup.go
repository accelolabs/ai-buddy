// Package cleanup logs best-effort resource cleanup failures.
package cleanup

import (
	"database/sql"
	"errors"
	"io"
	"log/slog"
)

func Close(resource io.Closer) {
	if err := resource.Close(); err != nil {
		slog.Warn("resource cleanup failed", "error", err)
	}
}

func Rollback(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		slog.Warn("resource cleanup failed", "error", err)
	}
}
