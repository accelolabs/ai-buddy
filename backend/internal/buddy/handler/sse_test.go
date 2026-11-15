package handler

import (
	"errors"
	"net/http/httptest"
	"testing"
)

type failingStreamWriter struct {
	*httptest.ResponseRecorder
	writeErr error
	flushErr error
}

func (w *failingStreamWriter) Write(body []byte) (int, error) {
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	return w.ResponseRecorder.Write(body)
}

func (w *failingStreamWriter) FlushError() error { return w.flushErr }

func TestSSEWriter(t *testing.T) {
	w := httptest.NewRecorder()
	stream := sseWriter{w: w}
	if err := stream.send("text", textResponse{Text: "Первая\nвторая"}); err != nil {
		t.Fatal(err)
	}
	if err := stream.send("source", nil); err != nil {
		t.Fatal(err)
	}
	if !stream.started || !w.Flushed || w.Header().Get("Content-Type") != "text/event-stream" || w.Header().Get("Cache-Control") != "no-cache" || w.Header().Get("X-Accel-Buffering") != "no" || w.Body.String() != "event: text\ndata: {\"text\":\"Первая\\nвторая\"}\n\nevent: source\ndata: null\n\n" {
		t.Fatalf("stream: %+v %s", w, w.Body.String())
	}
}

func TestSSEWriterErrors(t *testing.T) {
	errFailure := errors.New("stream failure")
	for _, stage := range []string{"json", "write", "flush"} {
		t.Run(stage, func(t *testing.T) {
			w := &failingStreamWriter{ResponseRecorder: httptest.NewRecorder()}
			stream := sseWriter{w: w}
			var data any = textResponse{Text: "answer"}
			switch stage {
			case "json":
				data = make(chan int)
			case "write":
				w.writeErr = errFailure
			case "flush":
				w.flushErr = errFailure
			}
			err := stream.send("text", data)
			if stage == "json" {
				if err == nil || stream.started || len(w.Header()) != 0 || w.Body.Len() != 0 {
					t.Fatal("JSON error started stream", err)
				}
			} else if !errors.Is(err, errFailure) || !stream.started {
				t.Fatal("lost stream error or start state", err)
			}
		})
	}
}
