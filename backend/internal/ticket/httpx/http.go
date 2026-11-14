package httpx

import (
	"errors"
	"net/http"

	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/ticket/apperror"
)

func Failure(w http.ResponseWriter, err error) {
	code, message := 500, "storage unavailable"
	switch {
	case errors.Is(err, apperror.ErrNoMentor):
		code, message = 422, apperror.ErrNoMentor.Error()
	case errors.Is(err, apperror.ErrInvalid), errors.Is(err, httpx.ErrInvalid):
		code, message = 400, "invalid input"
	case errors.Is(err, apperror.ErrNotFound):
		code, message = 404, "not found"
	case errors.Is(err, apperror.ErrConflict):
		code, message = 409, apperror.ErrConflict.Error()
	}

	httpx.JSON(w, code, map[string]string{"error": message})
}
