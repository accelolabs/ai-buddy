package tickets

import (
	"ai-buddy/backend-ticket/internal/apperror"
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeRepository struct {
	ticket             Ticket
	findErr, createErr error
	creates            int
}

func (f *fakeRepository) Find(context.Context, string, string) (Ticket, error) {
	return f.ticket, f.findErr
}
func (f *fakeRepository) Create(_ context.Context, s, r, k string, in CreateInput) (Ticket, bool, error) {
	f.creates++
	return Ticket{SenderID: s, RecipientID: r, Question: in.Question, Description: in.Description}, true, f.createErr
}
func (f *fakeRepository) Sent(context.Context, string) ([]Ticket, error) {
	return []Ticket{f.ticket}, f.findErr
}
func (f *fakeRepository) Incoming(context.Context, string) ([]Ticket, error) {
	return []Ticket{f.ticket}, f.findErr
}

type fakeFinder struct {
	id    string
	err   error
	calls int
}

func (f *fakeFinder) Mentor(context.Context, string) (string, error) { f.calls++; return f.id, f.err }
func TestCreate(t *testing.T) {
	down := errors.New("down")
	for _, tc := range []struct {
		name, key, q, d            string
		saved                      Ticket
		find, tracker, write, want error
		mentor                     string
		calls, creates             int
		created                    bool
	}{
		{name: "normalize", key: "k", q: " q \n", d: " d ", find: apperror.ErrNotFound, mentor: "m", calls: 1, creates: 1, created: true},
		{name: "empty key", key: " ", q: "q", d: "d", want: apperror.ErrInvalid},
		{name: "long key", key: strings.Repeat("я", 129), q: "q", d: "d", want: apperror.ErrInvalid},
		{name: "empty question", key: "k", q: " ", d: "d", want: apperror.ErrInvalid},
		{name: "empty description", key: "k", q: "q", want: apperror.ErrInvalid},
		{name: "repeat", key: "k", q: " q ", d: "d", saved: Ticket{Question: "q", Description: "d", RecipientID: "old"}},
		{name: "conflict", key: "k", q: "q", d: "d", want: apperror.ErrConflict},
		{name: "read failure", key: "k", q: "q", d: "d", find: down, want: down},
		{name: "unassigned", key: "k", q: "q", d: "d", find: apperror.ErrNotFound, calls: 1, want: apperror.ErrNoMentor},
		{name: "tracker failure", key: "k", q: "q", d: "d", find: apperror.ErrNotFound, tracker: apperror.ErrTracker, calls: 1, want: apperror.ErrTracker},
		{name: "unknown", key: "k", q: "q", d: "d", find: apperror.ErrNotFound, tracker: apperror.ErrNotFound, calls: 1, want: apperror.ErrNotFound},
		{name: "write failure", key: "k", q: "q", d: "d", find: apperror.ErrNotFound, mentor: "m", write: down, want: down, calls: 1, creates: 1, created: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{ticket: tc.saved, findErr: tc.find, createErr: tc.write}
			finder := &fakeFinder{id: tc.mentor, err: tc.tracker}
			got, created, err := New(repo, finder).Create(context.Background(), "s", tc.key, CreateInput{Question: tc.q, Description: tc.d})
			if !errors.Is(err, tc.want) || created != tc.created || finder.calls != tc.calls || repo.creates != tc.creates {
				t.Fatalf("%+v %v %v calls %d creates %d", got, created, err, finder.calls, repo.creates)
			}
			if tc.name == "normalize" && (got.Question != "q" || got.Description != "d" || got.SenderID != "s" || got.RecipientID != "m") {
				t.Fatal(got)
			}
			if tc.name == "repeat" && got.RecipientID != "old" {
				t.Fatal(got)
			}
		})
	}
}
func TestLists(t *testing.T) {
	s := New(&fakeRepository{ticket: Ticket{ID: "one"}}, nil)
	for _, list := range []func(context.Context, string) ([]Ticket, error){s.Sent, s.Incoming} {
		got, err := list(context.Background(), "s")
		if err != nil || len(got) != 1 {
			t.Fatal(got, err)
		}
	}
}
