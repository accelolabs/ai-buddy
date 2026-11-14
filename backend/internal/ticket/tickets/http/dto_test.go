package ticketshttp

import (
	"testing"
	"time"

	"ai-buddy/internal/testutil"
	"ai-buddy/internal/ticket/tickets"
)

func TestResponseContracts(t *testing.T) {
	ticket := tickets.Ticket{ID: "ticket", SenderID: "intern", RecipientID: "mentor", Question: "Question", Description: "Details", CreatedAt: time.Date(2026, 11, 14, 12, 30, 0, 0, time.UTC)}
	const want = `{"id":"ticket","sender_id":"intern","recipient_id":"mentor","question":"Question","description":"Details","created_at":"2026-11-14T12:30:00Z"}`
	testutil.JSON(t, fromTicket(ticket), want)
	testutil.JSON(t, fromTickets([]tickets.Ticket{ticket}), `{"tickets":[`+want+`]}`)
	testutil.JSON(t, fromTickets(nil), `{"tickets":[]}`)
	testutil.JSON(t, fromTickets([]tickets.Ticket{}), `{"tickets":[]}`)
}
