package tracker

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid input")
	ErrConflict = errors.New("idempotency key conflicts with existing task")
)

type Profile struct {
	ID           string  `json:"authentik_id"`
	Name         string  `json:"name"`
	Username     string  `json:"username"`
	Email        string  `json:"email"`
	Role         string  `json:"role"`
	Team         string  `json:"team"`
	StartDate    *string `json:"start_date"`
	Goals        string  `json:"goals"`
	Expectations string  `json:"expectations"`
}
type Task struct {
	ID          string     `json:"id"`
	InternID    string     `json:"intern_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Deadline    string     `json:"deadline"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Status      string     `json:"status"`
	Overdue     bool       `json:"overdue"`
}
type Input struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Deadline    string `json:"deadline"`
}
type Progress struct {
	Intern       Profile  `json:"intern"`
	Mentor       *Profile `json:"mentor"`
	StartDate    *string  `json:"start_date"`
	Goals        string   `json:"goals"`
	Expectations string   `json:"expectations"`
	Tasks        []Task   `json:"tasks"`
	Completed    int      `json:"completed"`
	Total        int      `json:"total"`
	Percent      int      `json:"percent"`
	NextTask     *Task    `json:"next_task"`
}
type Store interface {
	Intern(context.Context, string) (Profile, error)
	Mentor(context.Context, string) (*Profile, error)
	Interns(context.Context, string) ([]Profile, error)
	Assigned(context.Context, string, string) (bool, error)
	Tasks(context.Context, string) ([]Task, error)
	Create(context.Context, string, string, Input) (Task, bool, error)
	Complete(context.Context, string, string) (Task, error)
}
type Service struct {
	Store    Store
	Location *time.Location
	Now      func() time.Time
}

func New(store Store, loc *time.Location) *Service { return &Service{store, loc, time.Now} }
func (s *Service) decorate(t Task) Task {
	t.Status = "pending"
	if t.CompletedAt != nil {
		t.Status = "completed"
	}
	t.Overdue = t.CompletedAt == nil && t.Deadline < s.Now().In(s.Location).Format(time.DateOnly)
	return t
}
func (s *Service) Tasks(ctx context.Context, id string) ([]Task, error) {
	if _, e := s.Store.Intern(ctx, id); e != nil {
		return nil, e
	}
	ts, e := s.Store.Tasks(ctx, id)
	if e != nil {
		return nil, e
	}
	if ts == nil {
		ts = []Task{}
	}
	sort.SliceStable(ts, func(i, j int) bool {
		if ts[i].CreatedAt.Equal(ts[j].CreatedAt) {
			return ts[i].ID < ts[j].ID
		}
		return ts[i].CreatedAt.Before(ts[j].CreatedAt)
	})
	for i := range ts {
		ts[i] = s.decorate(ts[i])
	}
	return ts, nil
}
func (s *Service) Create(ctx context.Context, id, key string, in Input) (Task, bool, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.Deadline = strings.TrimSpace(in.Deadline)
	if strings.TrimSpace(key) == "" || len(key) > 256 || in.Title == "" {
		return Task{}, false, ErrInvalid
	}
	if _, e := time.Parse(time.DateOnly, in.Deadline); e != nil {
		return Task{}, false, ErrInvalid
	}
	if _, e := s.Store.Intern(ctx, id); e != nil {
		return Task{}, false, e
	}
	t, created, e := s.Store.Create(ctx, id, key, in)
	return s.decorate(t), created, e
}
func (s *Service) Complete(ctx context.Context, id, taskID string) (Task, error) {
	if _, e := s.Store.Intern(ctx, id); e != nil {
		return Task{}, e
	}
	t, e := s.Store.Complete(ctx, id, taskID)
	return s.decorate(t), e
}
func (s *Service) Mentor(ctx context.Context, id string) (*Profile, error) {
	if _, e := s.Store.Intern(ctx, id); e != nil {
		return nil, e
	}
	return s.Store.Mentor(ctx, id)
}
func (s *Service) Progress(ctx context.Context, id string) (Progress, error) {
	p, e := s.Store.Intern(ctx, id)
	if e != nil {
		return Progress{}, e
	}
	m, e := s.Store.Mentor(ctx, id)
	if e != nil {
		return Progress{}, e
	}
	ts, e := s.Tasks(ctx, id)
	if e != nil {
		return Progress{}, e
	}
	out := Progress{Intern: p, Mentor: m, StartDate: p.StartDate, Goals: p.Goals, Expectations: p.Expectations, Tasks: ts, Total: len(ts)}
	for i := range ts {
		t := &ts[i]
		if t.CompletedAt != nil {
			out.Completed++
			continue
		}
		if out.NextTask == nil || (!out.NextTask.Overdue && (t.Overdue || t.Deadline < out.NextTask.Deadline)) {
			out.NextTask = t
		}
	}
	if out.Total > 0 {
		out.Percent = 100 * out.Completed / out.Total
	}
	return out, nil
}
func (s *Service) MentorProgress(ctx context.Context, mentor, id string) (Progress, error) {
	ok, e := s.Store.Assigned(ctx, mentor, id)
	if e != nil {
		return Progress{}, e
	}
	if !ok {
		return Progress{}, ErrNotFound
	}
	return s.Progress(ctx, id)
}
