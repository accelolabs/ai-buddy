package wikihttp

import (
	"errors"
	"net/http"

	"ai-buddy/internal/platform/httpx"
	"ai-buddy/internal/wiki"
)

func fail(w http.ResponseWriter, err error) {
	code := 503
	message := "service unavailable"
	if errors.Is(err, wiki.ErrNotFound) {
		code = 404
		message = "not found"
	}
	httpx.JSON(w, code, map[string]string{"error": message})
}
