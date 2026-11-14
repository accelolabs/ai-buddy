package seed

import (
	"context"
	"database/sql"
)

// Run is intentionally empty. Add repeatable service-specific seed data after copying.
func Run(ctx context.Context, db *sql.DB) error { return ctx.Err() }
