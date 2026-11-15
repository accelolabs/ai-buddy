package tracker

import (
	"errors"
)

var (
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid input")
	ErrConflict = errors.New("idempotency key conflicts with existing task")
)
