package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

func Run(ctx context.Context, db *sql.DB, files fs.FS, command string) error {
	p, err := goose.NewProvider(goose.DialectPostgres, db, files)
	if err != nil {
		return err
	}

	switch command {
	case "up":
		_, err = p.Up(ctx)
	case "down":
		_, err = p.Down(ctx)
	case "status":
		statuses, statusErr := p.Status(ctx)
		if statusErr != nil {
			return statusErr
		}
		for _, status := range statuses {
			fmt.Printf("%s: %s\n", status.Source.Path, status.State)
		}
	default:
		return errors.New("command must be up, down or status")
	}

	return err
}
