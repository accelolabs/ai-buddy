package ticket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTicketWireContract(t *testing.T) {
	for _, handler := range []http.HandlerFunc{(&Service{repository: emptyLists{}}).SentHTTP, (&Service{repository: emptyLists{}}).IncomingHTTP} {
		w := httptest.NewRecorder()
		handler(
			w,
			httptest.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				"/",
				nil,
			),
		)

		if w.Code != 200 || w.Body.String() != `{"tickets":[]}`+"\n" {
			t.Fatal(w.Code, w.Body)
		}
	}

	data, err := json.Marshal(Ticket{})
	if err != nil ||
		string(
			data,
		) != `{"id":"","sender_id":"","recipient_id":"","question":"","description":"","created_at":"0001-01-01T00:00:00Z"}` {
		t.Fatalf("%s %v", data, err)
	}
}

type emptyLists struct{ Repository }

func (emptyLists) Sent(context.Context, string) ([]Ticket, error)     { return []Ticket{}, nil }
func (emptyLists) Incoming(context.Context, string) ([]Ticket, error) { return []Ticket{}, nil }
