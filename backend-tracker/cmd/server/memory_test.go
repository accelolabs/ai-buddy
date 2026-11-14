package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ai-buddy/backend-tracker/internal/apperror"
	"ai-buddy/backend-tracker/internal/interns"
	"ai-buddy/backend-tracker/internal/tasks"
)

type memory struct {
	profiles    map[string]interns.Profile
	assignments map[string]string
	tasks       []tasks.Task
	keys        map[string]tasks.CreateInput
	keyIDs      map[string]string
	fail        string
}

func newMemory() *memory {
	return &memory{profiles: map[string]interns.Profile{"i": {ID: "i", Role: "intern", Team: "backend"}, "j": {ID: "j", Role: "intern", Team: "platform"}, "m": {ID: "m", Role: "mentor", Team: "backend"}}, assignments: map[string]string{"i": "m"}, keys: map[string]tasks.CreateInput{}, keyIDs: map[string]string{}}
}

var broken = errors.New("SQL password=secret")

func (m *memory) Intern(_ context.Context, id string) (interns.Profile, error) {
	if m.fail == "intern" {
		return interns.Profile{}, broken
	}
	p, ok := m.profiles[id]
	if !ok || p.Role != "intern" {
		return interns.Profile{}, apperror.ErrNotFound
	}
	return p, nil
}
func (m *memory) Mentor(_ context.Context, id string) (*interns.Profile, error) {
	if m.fail == "mentor" {
		return nil, broken
	}
	v, ok := m.profiles[m.assignments[id]]
	if !ok {
		return nil, nil
	}
	return &v, nil
}
func (m *memory) Interns(_ context.Context, id string) ([]interns.Profile, error) {
	if m.fail == "interns" {
		return nil, broken
	}
	out := []interns.Profile{}
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
func (m *memory) Tasks(_ context.Context, id string) ([]tasks.Task, error) {
	if m.fail == "tasks" {
		return nil, broken
	}
	var out []tasks.Task
	for _, v := range m.tasks {
		if v.InternID == id {
			out = append(out, v)
		}
	}
	return out, nil
}
func (m *memory) Create(_ context.Context, id, key string, in tasks.CreateInput) (tasks.Task, bool, error) {
	if m.fail == "create" {
		return tasks.Task{}, false, broken
	}
	k := id + ":" + key
	if old, ok := m.keys[k]; ok {
		if old != in {
			return tasks.Task{}, false, apperror.ErrConflict
		}
		for _, v := range m.tasks {
			if v.ID == m.keyIDs[k] {
				return v, false, nil
			}
		}
	}
	t := tasks.Task{ID: fmt.Sprint(len(m.tasks) + 1), InternID: id, Title: in.Title, Description: in.Description, Deadline: in.Deadline, CreatedAt: time.Now()}
	m.tasks = append(m.tasks, t)
	m.keys[k] = in
	m.keyIDs[k] = t.ID
	return t, true, nil
}
func (m *memory) Complete(_ context.Context, id, task string) (tasks.Task, error) {
	if m.fail == "complete" {
		return tasks.Task{}, broken
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
	return tasks.Task{}, apperror.ErrNotFound
}
