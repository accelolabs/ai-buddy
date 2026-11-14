package wikihttp

import (
	"context"

	"ai-buddy/internal/wiki"
)

type Teams interface {
	Team(context.Context, string) (string, error)
}

type Handler struct {
	Service *wiki.Service
	Teams   Teams
}
