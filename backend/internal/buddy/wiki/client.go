package wiki

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"ai-buddy/internal/buddy/remote"
)

type Client struct {
	*remote.Client
	PublicURL string
}
type Source struct {
	DocumentID string `json:"document_id"`
	Title      string `json:"document_title"`
	Section    string `json:"section_title"`
	Text       string `json:"text"`
	URL        string `json:"url"`
}

func (c *Client) Search(ctx context.Context, id, query string) (out []Source, err error) {
	err = c.Do(ctx, "POST", "/internal/users/"+url.PathEscape(id)+"/search", "", map[string]string{"query": query}, &out)
	if err != nil {
		return nil, err
	}
	base, e := url.Parse(strings.TrimRight(c.PublicURL, "/") + "/")
	if e != nil {
		return nil, e
	}
	for i := range out {
		u, e := url.Parse(out[i].URL)
		if e != nil || u.IsAbs() || u.Host != "" || !strings.HasPrefix(u.Path, "/api/documents/") {
			return nil, fmt.Errorf("invalid Wiki source URL")
		}
		out[i].URL = base.ResolveReference(u).String()
	}
	if out == nil {
		out = []Source{}
	}
	return
}
