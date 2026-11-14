package buddyhttp

import (
	"encoding/json"

	"ai-buddy/internal/buddy"
)

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

type sendRequest struct {
	Content string `json:"content"`
}
type confirmRequest struct {
	Fields json.RawMessage `json:"fields"`
}
