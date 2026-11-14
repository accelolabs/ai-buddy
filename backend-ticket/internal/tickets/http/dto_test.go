package ticketshttp

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"ai-buddy/backend-ticket/internal/tickets"
)

func TestTicketResponseContract(t *testing.T) {
	ticket := tickets.Ticket{
		ID:          "ticket",
		SenderID:    "intern",
		RecipientID: "mentor",
		Question:    "Question",
		Description: "Details",
		CreatedAt:   time.Date(2026, 11, 14, 12, 30, 0, 0, time.FixedZone("MSK", 3*60*60)),
	}
	const ticketJSON = `{"id":"ticket","sender_id":"intern","recipient_id":"mentor","question":"Question","description":"Details","created_at":"2026-11-14T12:30:00+03:00"}`
	for _, tc := range []struct {
		name     string
		response any
		want     string
	}{
		{"single", fromTicket(ticket), ticketJSON},
		{"list", fromTickets([]tickets.Ticket{ticket}), `{"tickets":[` + ticketJSON + `]}`},
		{"nil list", fromTickets(nil), `{"tickets":[]}`},
		{"empty list", fromTickets([]tickets.Ticket{}), `{"tickets":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.response)
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %s\nwant %s", raw, tc.want)
			}
		})
	}
}
