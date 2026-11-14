package apperror

import "errors"

var (
	//nolint:staticcheck // Public API error text is preserved for existing clients.
	ErrNoMentor = errors.New("Ментор не назначен")
	ErrTracker  = errors.New("tracker unavailable")
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid input")
	ErrConflict = errors.New("idempotency key conflicts with existing ticket")
)
