package ticketshttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-buddy/internal/ticket/tickets"
)

func TestTicketWireContract(t *testing.T) {
	for _, handler := range []http.HandlerFunc{New(nilLists{}).Sent, New(nilLists{}).Incoming} {
		w := httptest.NewRecorder()
		handler(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 200 || w.Body.String() != `{"tickets":[]}`+"\n" {
			t.Fatal(w.Code, w.Body)
		}
	}

	data, err := json.Marshal(tickets.Ticket{})
	if err != nil || string(data) != `{"id":"","sender_id":"","recipient_id":"","question":"","description":"","created_at":"0001-01-01T00:00:00Z"}` {
		t.Fatalf("%s %v", data, err)
	}
}

type nilLists struct{ Service }

func (nilLists) Sent(context.Context, string) ([]tickets.Ticket, error)     { return nil, nil }
func (nilLists) Incoming(context.Context, string) ([]tickets.Ticket, error) { return nil, nil }
