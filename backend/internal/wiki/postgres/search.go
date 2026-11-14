package postgres

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"ai-buddy/internal/wiki"
)

// Search keeps both candidate branches and index fingerprint validation in one database snapshot.
func (r *Repository) Search(ctx context.Context, team, query string, embedding []float64, minimum float64, fingerprint string) ([]wiki.Result, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var stored string
	if err = tx.QueryRowContext(ctx, "SELECT fingerprint FROM wiki_state WHERE singleton").Scan(&stored); err != nil {
		return nil, err
	}
	if stored != fingerprint {
		return nil, wiki.ErrUnavailable
	}

	rows, err := tx.QueryContext(
		ctx, searchQuery,
		team,
		query,
		vector(embedding),
		minimum,
	)
	if err != nil {
		return nil, err
	}

	results := []wiki.Result{}
	for rows.Next() {
		var result wiki.Result
		if err := rows.Scan(
			&result.DocumentID,
			&result.DocumentTitle,
			&result.SectionTitle,
			&result.Anchor,
			&result.Text,
		); err != nil {
			rows.Close()
			return nil, err
		}
		results = append(results, result)
	}
	err = rows.Err()
	rows.Close()
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
