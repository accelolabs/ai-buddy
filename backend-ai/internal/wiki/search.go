package wiki

import (
	"backend-ai/internal/platform/cleanup"
	"backend-ai/internal/platform/database"
	"context"
	"database/sql"
	"strconv"
	"strings"
)

// Search keeps both candidate branches and index fingerprint validation in one database snapshot.
func (r *Store) Search(
	ctx context.Context,
	team, query string,
	embedding []float64,
	minimum float64,
	fingerprint string,
) ([]Source, error) {
	tx, err := r.DB.BeginTx(
		ctx,
		&sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead},
	)
	if err != nil {
		return nil, err
	}
	defer cleanup.Rollback(tx)

	var stored string
	if err := tx.QueryRowContext(ctx, "SELECT fingerprint FROM wiki_state WHERE singleton").
		Scan(&stored); err != nil {
		return nil, err
	}

	if stored != fingerprint {
		return nil, ErrUnavailable
	}

	results, err := database.Query(
		ctx,
		tx,
		searchQuery,
		scanSource,
		team,
		query,
		vector(embedding),
		minimum,
	)
	if err != nil {
		return nil, err
	}

	return results, tx.Commit()
}

func vector(embedding []float64) string {
	p := make([]string, len(embedding))
	for i, value := range embedding {
		p[i] = strconv.FormatFloat(value, 'g', -1, 64)
	}

	return "[" + strings.Join(p, ",") + "]"
}

func scanSource(row database.RowScanner) (result Source, err error) {
	err = row.Scan(
		&result.DocumentID,
		&result.Title,
		&result.Section,
		&result.Anchor,
		&result.Text,
	)
	result.URL = "/api/documents/" + result.DocumentID

	return result, err
}
