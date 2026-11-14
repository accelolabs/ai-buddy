package buddyhttp

import (
	"errors"
	"net/http"

	"ai-buddy/internal/buddy"
	"ai-buddy/internal/platform/auth"
	"ai-buddy/internal/platform/httpx"
)

func fail(w http.ResponseWriter, e error) {
	code := 500
	msg := "service unavailable"
	switch {
	case errors.Is(e, buddy.ErrNotFound):
		code = 404
		msg = e.Error()
	case errors.Is(e, buddy.ErrConflict):
		code = 409
		msg = e.Error()
	case errors.Is(e, buddy.ErrInvalid):
		code = 400
		msg = e.Error()
	}
	httpx.JSON(w, code, map[string]string{"error": msg})
}
func owner(r *http.Request) auth.User { u, _ := auth.CurrentUser(r.Context()); return u }
