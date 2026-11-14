package buddyhttp

import (
	"time"

	"ai-buddy/internal/buddy"
)

type Handler struct {
	Service   *buddy.Service
	Poll      time.Duration
	Heartbeat time.Duration
}

func New(s *buddy.Service) *Handler { return &Handler{s, 250 * time.Millisecond, 15 * time.Second} }
