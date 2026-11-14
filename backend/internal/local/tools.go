// Package local connects Buddy tools to application services in the same process.
package local

import (
	"ai-buddy/internal/buddy/wiki"
	"ai-buddy/internal/ticket/tickets"
	"ai-buddy/internal/tracker/postgres"
	"ai-buddy/internal/tracker/progress"
	"ai-buddy/internal/tracker/tasks"
	"ai-buddy/internal/users"
	wikidomain "ai-buddy/internal/wiki"
	"context"
	"encoding/json"
)

type Mentors struct{ Repository *postgres.Repository }

func (m Mentors) Mentor(ctx context.Context, id string) (string, error) {
	p, err := m.Repository.Mentor(ctx, id)
	if err != nil || p == nil {
		return "", err
	}
	return p.ID, nil
}

type Tracker struct {
	Tasks           *tasks.Service
	ProgressService *progress.Service
}

func (t Tracker) Progress(ctx context.Context, id string) (progress.Progress, error) {
	return t.ProgressService.Progress(ctx, id)
}
func (t Tracker) Create(ctx context.Context, id, key string, fields json.RawMessage) (json.RawMessage, error) {
	var in tasks.CreateInput
	if err := json.Unmarshal(fields, &in); err != nil {
		return nil, err
	}
	v, _, err := t.Tasks.Create(ctx, id, key, in)
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}
func (t Tracker) Complete(ctx context.Context, id, task string) (json.RawMessage, error) {
	v, err := t.Tasks.Complete(ctx, id, task)
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

type Ticket struct{ Service *tickets.Service }

func (t Ticket) Create(ctx context.Context, id, key string, fields json.RawMessage) (json.RawMessage, error) {
	var in tickets.CreateInput
	if err := json.Unmarshal(fields, &in); err != nil {
		return nil, err
	}
	v, _, err := t.Service.Create(ctx, id, key, in)
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}
func (t Ticket) List(ctx context.Context, id string) (json.RawMessage, error) {
	v, err := t.Service.Sent(ctx, id)
	if err != nil {
		return nil, err
	}
	if v == nil {
		v = []tickets.Ticket{}
	}
	return json.Marshal(v)
}

type Wiki struct {
	Service  *wikidomain.Service
	Profiles users.Reader
}

func (w Wiki) Search(ctx context.Context, id, query string) ([]wiki.Source, error) {
	u, err := w.Profiles.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	found, err := w.Service.Search(ctx, u.Team, query)
	if err != nil {
		return nil, err
	}
	out := []wiki.Source{}
	for _, v := range found {
		out = append(out, wiki.Source{DocumentID: v.DocumentID, Title: v.DocumentTitle, Section: v.SectionTitle, Text: v.Text, URL: "/api/documents/" + v.DocumentID})
	}
	return out, nil
}
