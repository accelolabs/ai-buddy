package seed

import (
	"context"
	"testing"
	"time"

	"ai-buddy/backend-ticket/internal/testdb"
)

func TestPostgresSeedIntegration(t *testing.T) {
	db := testdb.Open(t)
	srv := api(t, "")
	defer srv.Close()
	ctx := context.Background()
	runSeed := func() {
		t.Helper()
		if err := run(ctx, db, Config{URL: srv.URL, Token: "token"}, srv.Client(), finder{}, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	runSeed()
	if _, err := db.ExecContext(ctx, "UPDATE tickets SET question='live question', recipient_id='live mentor'"); err != nil {
		t.Fatal(err)
	}
	runSeed()
	var count int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM tickets WHERE question='live question' AND recipient_id='live mentor'").Scan(&count); err != nil || count != 3 {
		t.Fatal(count, err)
	}
	var total int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM tickets").Scan(&total); err != nil || total != 3 {
		t.Fatal(total, err)
	}
	// Force a failure on the second insert and verify the first is rolled back.
	if _, err := db.ExecContext(ctx, "DELETE FROM tickets"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE tickets ADD CONSTRAINT reject_second CHECK (sender_id <> '3')`); err != nil {
		t.Fatal(err)
	}
	if err := run(ctx, db, Config{URL: srv.URL, Token: "token"}, srv.Client(), finder{}, time.Now()); err == nil {
		t.Fatal("expected constraint error")
	}
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM tickets").Scan(&total); err != nil || total != 0 {
		t.Fatal(total, err)
	}
}
