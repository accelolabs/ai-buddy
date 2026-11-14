package buddyhttp

import (
	"encoding/json"
	"time"

	"ai-buddy/internal/buddy"
)

type chatDTO struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

func chat(c buddy.Chat) chatDTO { return chatDTO{c.ID, c.Title, c.CreatedAt} }

type runDTO struct {
	ID        string `json:"id"`
	ChatID    string `json:"chat_id"`
	Status    string `json:"status"`
	Error     string `json:"error"`
	EventsURL string `json:"events_url"`
}

func run(r buddy.Run) runDTO {
	return runDTO{r.ID, r.ChatID, r.Status, r.Error, "/api/runs/" + r.ID + "/events"}
}

type actionDTO struct {
	ID     string          `json:"id"`
	RunID  string          `json:"run_id"`
	Kind   string          `json:"kind"`
	Status string          `json:"status"`
	Fields json.RawMessage `json:"fields"`
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

func action(a buddy.Action) actionDTO {
	return actionDTO{a.ID, a.RunID, a.Kind, a.Status, a.Fields, a.Result, a.Error}
}

type messageDTO struct {
	ID        string    `json:"id"`
	RunID     string    `json:"run_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type cardDTO struct {
	RunID string          `json:"run_id"`
	Kind  string          `json:"kind"`
	Data  json.RawMessage `json:"data"`
}

type sendRequest struct {
	Content string `json:"content"`
}
type confirmRequest struct {
	Fields json.RawMessage `json:"fields"`
}
