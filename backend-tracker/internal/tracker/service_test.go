package tracker

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type memory struct {
	profiles    map[string]Profile
	assignments map[string]string
	tasks       []Task
	keys        map[string]Input
	keyIDs      map[string]string
	fail        string
}

func fixture() (*Service, *memory) {
	m := &memory{profiles: map[string]Profile{"i": {ID: "i", Role: "intern", Team: "backend"}, "j": {ID: "j", Role: "intern", Team: "platform"}, "m": {ID: "m", Role: "mentor", Team: "backend"}}, assignments: map[string]string{"i": "m"}, keys: map[string]Input{}, keyIDs: map[string]string{}}
	loc, _ := time.LoadLocation("Europe/Moscow")
	s := New(m, loc)
	s.Now = func() time.Time { return time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC) }
	return s, m
}

var broken = errors.New("SQL password=secret")

func (m *memory) Intern(_ context.Context, id string) (Profile, error) {
	if m.fail == "intern" {
		return Profile{}, broken
	}
	p, ok := m.profiles[id]
	if !ok || p.Role != "intern" {
		return Profile{}, ErrNotFound
	}
	return p, nil
}
func (m *memory) Mentor(_ context.Context, id string) (*Profile, error) {
	if m.fail == "mentor" {
		return nil, broken
	}
	v, ok := m.profiles[m.assignments[id]]
	if !ok {
		return nil, nil
	}
	return &v, nil
}
func (m *memory) Interns(_ context.Context, id string) ([]Profile, error) {
	if m.fail == "interns" {
		return nil, broken
	}
	out := []Profile{}
	for intern, mentor := range m.assignments {
		if mentor == id {
			out = append(out, m.profiles[intern])
		}
	}
	return out, nil
}
func (m *memory) Assigned(_ context.Context, mentor, id string) (bool, error) {
	if m.fail == "assigned" {
		return false, broken
	}
	return m.assignments[id] == mentor, nil
}
func (m *memory) Tasks(_ context.Context, id string) ([]Task, error) {
	if m.fail == "tasks" {
		return nil, broken
	}
	var out []Task
	for _, v := range m.tasks {
		if v.InternID == id {
			out = append(out, v)
		}
	}
	return out, nil
}
func (m *memory) Create(_ context.Context, id, key string, in Input) (Task, bool, error) {
	if m.fail == "create" {
		return Task{}, false, broken
	}
	k := id + ":" + key
	if old, ok := m.keys[k]; ok {
		if old != in {
			return Task{}, false, ErrConflict
		}
		for _, v := range m.tasks {
			if v.ID == m.keyIDs[k] {
				return v, false, nil
			}
		}
	}
	t := Task{ID: fmt.Sprint(len(m.tasks) + 1), InternID: id, Title: in.Title, Description: in.Description, Deadline: in.Deadline, CreatedAt: time.Now()}
	m.tasks = append(m.tasks, t)
	m.keys[k] = in
	m.keyIDs[k] = t.ID
	return t, true, nil
}
func (m *memory) Complete(_ context.Context, id, task string) (Task, error) {
	if m.fail == "complete" {
		return Task{}, broken
	}
	for i, t := range m.tasks {
		if t.ID == task && t.InternID == id {
			if t.CompletedAt == nil {
				now := time.Now()
				m.tasks[i].CompletedAt = &now
			}
			return m.tasks[i], nil
		}
	}
	return Task{}, ErrNotFound
}
func TestProgress(t *testing.T) {
	s, m := fixture()
	ctx := context.Background()
	p, e := s.Progress(ctx, "i")
	if e != nil || p.Total != 0 || p.Percent != 0 || p.NextTask != nil || p.Tasks == nil || p.Mentor.ID != "m" {
		t.Fatalf("empty: %+v %v", p, e)
	}
	p, e = s.Progress(ctx, "j")
	if e != nil || p.Mentor != nil {
		t.Fatal("unassigned", p, e)
	}
	now := s.Now()
	done := now
	m.tasks = []Task{
		{ID: "z", InternID: "i", Deadline: "2026-11-13", CreatedAt: now.Add(time.Hour)},
		{ID: "b", InternID: "i", Deadline: "2026-11-12", CreatedAt: now},
		{ID: "a", InternID: "i", Deadline: "2026-11-13", CreatedAt: now},
		{ID: "c", InternID: "i", Deadline: "2026-11-01", CreatedAt: now, CompletedAt: &done},
	}
	p, e = s.Progress(ctx, "i")
	if e != nil || p.Completed != 1 || p.Total != 4 || p.Percent != 25 || p.NextTask.ID != "a" || !p.NextTask.Overdue {
		t.Fatalf("overdue order: %+v %v", p, e)
	}
	for i := range m.tasks {
		m.tasks[i].CompletedAt = nil
		m.tasks[i].Deadline = "2026-11-15"
	}
	m.tasks[0].Deadline = "2026-11-14"
	p, _ = s.Progress(ctx, "i")
	if p.NextTask.ID != "z" || p.NextTask.Overdue {
		t.Fatal("nearest", p)
	}
	m.tasks[0].Deadline = "2026-11-15"
	p, _ = s.Progress(ctx, "i")
	if p.NextTask.ID != "a" {
		t.Fatal("deadline ties", p)
	}
	for i := range m.tasks {
		m.tasks[i].CompletedAt = &done
	}
	p, _ = s.Progress(ctx, "i")
	if p.Percent != 100 || p.NextTask != nil {
		t.Fatal("completed", p)
	}
}
func TestCalendarBoundary(t *testing.T) {
	s, _ := fixture()
	v := Task{Deadline: "2026-11-14"}
	s.Now = func() time.Time { return time.Date(2026, 11, 14, 20, 59, 59, 0, time.UTC) }
	if s.decorate(v).Overdue {
		t.Fatal("today overdue")
	}
	s.Now = func() time.Time { return time.Date(2026, 11, 14, 21, 0, 0, 0, time.UTC) }
	if !s.decorate(v).Overdue {
		t.Fatal("local day did not change")
	}
}
func TestValidationAndOwnership(t *testing.T) {
	s, _ := fixture()
	ctx := context.Background()
	for _, tc := range []struct {
		key string
		in  Input
	}{{"", Input{Title: "a", Deadline: "2026-11-01"}}, {"key", Input{Title: "  ", Deadline: "2026-11-01"}}, {"key", Input{Title: "a", Deadline: "2026-02-30"}}, {"key", Input{Title: "a", Deadline: "2026-1-01"}}} {
		if _, _, e := s.Create(ctx, "i", tc.key, tc.in); e != ErrInvalid {
			t.Fatal("validation", e)
		}
	}
	in := Input{Title: "  task  ", Description: " details ", Deadline: " 2020-01-01 "}
	v, created, e := s.Create(ctx, "i", "key", in)
	if e != nil || !created || v.Title != "task" || v.Description != "details" || !v.Overdue {
		t.Fatal(v, created, e)
	}
	again, created, e := s.Create(ctx, "i", "key", in)
	if e != nil || created || again.ID != v.ID {
		t.Fatal("retry", e)
	}
	in.Title = "other"
	if _, _, e = s.Create(ctx, "i", "key", in); e != ErrConflict {
		t.Fatal("conflict", e)
	}
	if _, e = s.Complete(ctx, "j", v.ID); e != ErrNotFound {
		t.Fatal("ownership", e)
	}
	v, e = s.Complete(ctx, "i", v.ID)
	again, e2 := s.Complete(ctx, "i", v.ID)
	if e != nil || e2 != nil || v.Status != "completed" || v.Overdue || !again.CompletedAt.Equal(*v.CompletedAt) {
		t.Fatal("completion", v, again, e, e2)
	}
	if _, e = s.MentorProgress(ctx, "m", "j"); e != ErrNotFound {
		t.Fatal("assignment", e)
	}
	if _, e = s.MentorProgress(ctx, "m", "i"); e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{"unknown", "m"} {
		if _, e = s.Tasks(ctx, id); e != ErrNotFound {
			t.Fatal("unknown intern")
		}
		if _, _, e = s.Create(ctx, id, "key", in); e != ErrNotFound {
			t.Fatal(e)
		}
		if _, e = s.Complete(ctx, id, v.ID); e != ErrNotFound {
			t.Fatal(e)
		}
		if _, e = s.Mentor(ctx, id); e != ErrNotFound {
			t.Fatal(e)
		}
		if _, e = s.Progress(ctx, id); e != ErrNotFound {
			t.Fatal(e)
		}
	}
}
func TestServiceErrors(t *testing.T) {
	s, m := fixture()
	ctx := context.Background()
	for _, fail := range []string{"intern", "mentor", "tasks"} {
		m.fail = fail
		if _, e := s.Progress(ctx, "i"); e != broken {
			t.Fatal(fail, e)
		}
	}
	m.fail = "mentor"
	if _, e := s.Mentor(ctx, "i"); e != broken {
		t.Fatal(e)
	}
	m.fail = "assigned"
	if _, e := s.MentorProgress(ctx, "m", "i"); e != broken {
		t.Fatal(e)
	}
	m.fail = "tasks"
	if _, e := s.Tasks(ctx, "i"); e != broken {
		t.Fatal(e)
	}
	m.fail = "create"
	if _, _, e := s.Create(ctx, "i", "k", Input{Title: "a", Deadline: "2026-01-01"}); e != broken {
		t.Fatal(e)
	}
	m.fail = "complete"
	if _, e := s.Complete(ctx, "i", "x"); e != broken {
		t.Fatal(e)
	}
}
