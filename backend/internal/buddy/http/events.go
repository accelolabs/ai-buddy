package buddyhttp

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ai-buddy/internal/buddy"
)

func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	after := int64(0)
	if raw := r.Header.Get("Last-Event-ID"); raw != "" {
		var e error
		after, e = strconv.ParseInt(raw, 10, 64)
		if e != nil || after < 0 {
			fail(w, buddy.ErrInvalid)
			return
		}
	}
	events, current, e := h.Service.Repo.Events(r.Context(), owner(r).ID, r.PathValue("id"), after)
	if e != nil {
		fail(w, e)
		return
	}
	if _, ok := w.(http.Flusher); !ok {
		fail(w, errors.New("streaming unavailable"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	controller := http.NewResponseController(w)
	write := func(text string) bool {
		if err := controller.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil && !errors.Is(err, http.ErrNotSupported) {
			return false
		}
		if _, e := fmt.Fprint(w, text); e != nil {
			return false
		}
		return controller.Flush() == nil
	}
	poll := time.NewTicker(h.Poll)
	defer poll.Stop()
	heartbeat := time.NewTicker(h.Heartbeat)
	defer heartbeat.Stop()
	for {
		for _, event := range events {
			if !write(fmt.Sprintf("id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Kind, event.Data)) {
				return
			}
			after = event.ID
		}
		if !current.Active() {
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if !write(": heartbeat\n\n") {
				return
			}
		case <-poll.C:
		}
		events, current, e = h.Service.Repo.Events(r.Context(), owner(r).ID, current.ID, after)
		if e != nil {
			return
		}
	}
}
