package seed

import (
	"ai-buddy/internal/migrations"
	"ai-buddy/internal/testutil"
	"context"
	"testing"
)

func TestTransactionalSeedAndPreservation(t *testing.T) {
	db := testutil.Database(t, migrations.Files)
	ctx := context.Background()
	if _, err := db.Exec(`ALTER TABLE tasks ADD CONSTRAINT reject_seed CHECK(title <> 'Познакомиться с командой')`); err != nil {
		t.Fatal(err)
	}
	if err := Run(ctx, db); err == nil {
		t.Fatal("expected transactional seed failure")
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM profiles`).Scan(&count); err != nil || count != 0 {
		t.Fatal("partial seed survived", count, err)
	}
	if _, err := db.Exec(`ALTER TABLE tasks DROP CONSTRAINT reject_seed`); err != nil {
		t.Fatal(err)
	}
	if err := Run(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		table string
		count int
	}{{"profiles", 6}, {"assignments", 3}, {"tasks", 12}, {"tickets", 3}} {
		if err := db.QueryRow(`SELECT count(*) FROM ` + tc.table).Scan(&count); err != nil || count != tc.count {
			t.Fatal(tc.table, count, err)
		}
	}
	if _, err := db.Exec(`UPDATE profiles SET name='Edited name' WHERE id='intern.backend.1'; UPDATE tasks SET title='Edited task' WHERE id='demo:intern.backend.1:0'; UPDATE tickets SET description='Edited ticket' WHERE id='demo:ticket:intern.backend.1'`); err != nil {
		t.Fatal(err)
	}
	if err := Run(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ query, want string }{
		{`SELECT name FROM profiles WHERE id='intern.backend.1'`, "Edited name"},
		{`SELECT title FROM tasks WHERE id='demo:intern.backend.1:0'`, "Edited task"},
		{`SELECT description FROM tickets WHERE id='demo:ticket:intern.backend.1'`, "Edited ticket"},
		{`SELECT name FROM profiles WHERE id='mentor.backend'`, "Алексей Смирнов"},
	} {
		var actual string
		if err := db.QueryRow(tc.query).Scan(&actual); err != nil || actual != tc.want {
			t.Fatal(actual, err)
		}
	}
}
