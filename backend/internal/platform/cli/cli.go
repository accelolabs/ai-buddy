package cli

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"ai-buddy/internal/platform/config"
	"ai-buddy/internal/platform/database"
)

type Operation func(context.Context, *sql.DB) error
type Prepare func(service, command string) (Operation, error)

// Run validates the entire invocation before reading secrets or opening a database.
func Run(args []string, migration bool, prepare Prepare, open func(context.Context, string) (*sql.DB, error)) error {
	if len(args) < 1 || (args[0] != "tracker" && args[0] != "ticket" && !(migration && (args[0] == "wiki" || args[0] == "buddy"))) {
		return errors.New("service must be tracker, ticket, wiki or buddy (migrations only)")
	}
	command := "up"
	if migration {
		if len(args) > 2 {
			return errors.New("usage: migrate <tracker|ticket|wiki|buddy> [up|down|status]")
		}
		if len(args) == 2 {
			command = args[1]
		}
		if command != "up" && command != "down" && command != "status" {
			return errors.New("command must be up, down or status")
		}
	} else if len(args) != 1 {
		return errors.New("usage: seed <tracker|ticket>")
	}
	op, err := prepare(args[0], command)
	if err != nil {
		return err
	}
	c, err := config.LoadDatabase()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, err := open(ctx, c.URL)
	if err != nil {
		return err
	}
	defer db.Close()
	return op(ctx, db)
}

var Open = database.Open
