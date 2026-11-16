package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type sseWriter struct {
	w       http.ResponseWriter
	started bool
}

func (s *sseWriter) send(event string, data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if !s.started {
		s.w.Header().Set("Content-Type", "text/event-stream")
		s.w.Header().Set("Cache-Control", "no-cache")
		s.w.Header().Set("X-Accel-Buffering", "no")
		s.started = true
	}

	if _, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, body); err != nil {
		return err
	}

	return http.NewResponseController(s.w).Flush()
}
