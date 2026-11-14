package tracker

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"ai-buddy/internal/buddy/remote"
)

type Client struct{ *remote.Client }
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Status      string     `json:"status"`
	Deadline    string     `json:"deadline"`
	Overdue     bool       `json:"overdue"`
	CompletedAt *time.Time `json:"completed_at"`
	Description string     `json:"description"`
}
type Progress struct {
	Tasks        []Task          `json:"tasks"`
	Mentor       json.RawMessage `json:"mentor"`
	Total        int             `json:"total"`
	Completed    int             `json:"completed"`
	Percent      int             `json:"percent"`
	Goals        string          `json:"goals"`
	Expectations string          `json:"expectations"`
	StartDate    *string         `json:"start_date"`
	NextTask     *Task           `json:"next_task"`
}

func path(id string) string { return "/internal/interns/" + url.PathEscape(id) }
func (c *Client) Progress(ctx context.Context, id string) (p Progress, err error) {
	err = c.Do(ctx, "GET", path(id)+"/progress", "", nil, &p)
	return
}
func (c *Client) Create(ctx context.Context, id, key string, fields json.RawMessage) (r json.RawMessage, err error) {
	err = c.Do(ctx, "POST", path(id)+"/tasks", key, fields, &r)
	return
}
func (c *Client) Complete(ctx context.Context, id, task string) (r json.RawMessage, err error) {
	err = c.Do(ctx, "POST", path(id)+"/tasks/"+url.PathEscape(task)+"/complete", "", nil, &r)
	return
}
