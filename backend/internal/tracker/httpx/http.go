package httpx

import (
	"errors"
	"net/http"

	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/tracker/apperror"
)

var JSON = httpx.JSON
var Fallbacks = httpx.Fallbacks
var KeyAuth = httpx.KeyAuth
var Owner = httpx.Owner

func Decode(w http.ResponseWriter, r *http.Request, target any) error {
	if httpx.Decode(w, r, target) != nil {
		return apperror.ErrInvalid
	}
	return nil
}
func Failure(w http.ResponseWriter, err error) {
	code, message := 500, "storage unavailable"
	switch {
	case errors.Is(err, apperror.ErrInvalid):
		code, message = 400, "invalid input"
	case errors.Is(err, apperror.ErrNotFound):
		code, message = 404, "not found"
	case errors.Is(err, apperror.ErrConflict):
		code, message = 409, apperror.ErrConflict.Error()
	}

	JSON(w, code, map[string]string{"error": message})
}
