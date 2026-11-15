package wiki

import (
	"database/sql"
)

type Store struct {
	DB *sql.DB
}
