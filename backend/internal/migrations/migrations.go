package migrations

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3"
)

func Run(ctx context.Context, db *sql.DB, command string) error {
	p, err := goose.NewProvider(goose.DialectPostgres, db, Files)
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

//go:embed *.sql
var Files embed.FS
