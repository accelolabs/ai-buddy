package ticket

import (
	"ai-buddy/internal/users"
	"context"
	"errors"
	"strings"
	"testing"
)

type dependencies struct {
	saved              Ticket
	findErr, mentorErr error
	mentor             string
	lookups, writes    int
}

func (d *dependencies) Find(context.Context, string, string) (Ticket, error) {
	return d.saved, d.findErr
}

func (d *dependencies) Create(
	_ context.Context,
	sender, recipient, _ string,
	in CreateInput,
) (Ticket, bool, error) {
	d.writes++

	return Ticket{
		SenderID:    sender,
		RecipientID: recipient,
		Question:    in.Question,
		Description: in.Description,
	}, true, nil
}
func (*dependencies) Sent(context.Context, string) ([]Ticket, error)     { return nil, nil }
func (*dependencies) Incoming(context.Context, string) ([]Ticket, error) { return nil, nil }
func (d *dependencies) Mentor(context.Context, string) (*users.User, error) {
	d.lookups++
	if d.mentor == "" {
		return nil, d.mentorErr
	}

	return &users.User{ID: d.mentor}, d.mentorErr
}

func TestReplayAndMentorPolicy(t *testing.T) {
	d := &dependencies{
		saved: Ticket{
			ID:          "existing",
			RecipientID: "old",
			Question:    "Q",
			Description: "D",
		},
		mentorErr: errors.New("repository unavailable"),
	}
	s := New(d, d)
	ctx := context.Background()
	in := CreateInput{Question: " Q ", Description: " D "}

	got, fresh, err := s.Create(ctx, "i", "key", in)
	if err != nil || fresh || got.ID != "existing" || got.RecipientID != "old" ||
		d.lookups != 0 {
		t.Fatal(got, fresh, err, d)
	}

	if _, _, err = s.Create(
		ctx,
		"i",
		"key",
		CreateInput{Question: "changed", Description: "D"},
	); !errors.Is(err, ErrConflict) ||
		d.lookups != 0 {
		t.Fatal(err, d)
	}

	d.findErr = ErrNotFound
	for _, failure := range []error{errors.New("repository unavailable"), ErrNotFound} {
		d.mentorErr = failure
		if _, _, err = s.Create(
			ctx,
			"i",
			"new",
			in,
		); !errors.Is(err, failure) ||
			d.writes != 0 {
			t.Fatal(err, d)
		}
	}

	d.mentorErr = nil

	if _, _, err = s.Create(ctx, "i", "new", in); !errors.Is(err, ErrNoMentor) {
		t.Fatal(err)
	}

	d.mentor = "reassigned"

	got, fresh, err = s.Create(ctx, "i", strings.Repeat("я", 128), in)
	if err != nil || !fresh || got.RecipientID != "reassigned" || got.Question != "Q" ||
		got.Description != "D" {
		t.Fatal(got, err)
	}

	before := d.lookups

	for _, key := range []string{" ", strings.Repeat("я", 129)} {
		if _, _, err = s.Create(
			ctx,
			"i",
			key,
			in,
		); !errors.Is(err, ErrInvalid) ||
			d.lookups != before {
			t.Fatal(err)
		}
	}
}
