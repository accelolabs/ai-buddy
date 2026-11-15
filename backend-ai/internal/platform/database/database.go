// Package database opens PostgreSQL pools and consumes typed query results.
package database

import (
	"backend-ai/internal/platform/cleanup"
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Register the pgx database/sql driver.
)

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err = db.PingContext(ctx); err != nil {
		cleanup.Close(db)
		return nil, err
	}

	return db, nil
}
