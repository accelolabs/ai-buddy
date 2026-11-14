package ticket

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"ai-buddy/internal/buddy/remote"
)

type Client struct{ *remote.Client }

func (c *Client) Create(ctx context.Context, id, key string, fields json.RawMessage) (r json.RawMessage, err error) {
	err = c.Do(ctx, "POST", "/internal/interns/"+url.PathEscape(id)+"/tickets", key, fields, &r)
	if err == nil {
		var created struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(r, &created) != nil || strings.TrimSpace(created.ID) == "" {
			err = errors.New("invalid ticket creation response")
		}
	}
	return
}
func (c *Client) List(ctx context.Context, id string) (r json.RawMessage, err error) {
	err = c.Do(ctx, "GET", "/internal/interns/"+url.PathEscape(id)+"/tickets", "", nil, &r)
	return
}
