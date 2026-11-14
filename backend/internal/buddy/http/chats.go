package buddyhttp

import (
	"net/http"

	"ai-buddy/internal/platform/httpx"
)

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	c, e := h.Service.Repo.CreateChat(r.Context(), owner(r).ID)
	if e != nil {
		fail(w, e)
		return
	}
	httpx.JSON(w, 201, chat(c))
}
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	all, e := h.Service.Repo.Chats(r.Context(), owner(r).ID)
	if e != nil {
		fail(w, e)
		return
	}
	out := []chatDTO{}
	for _, c := range all {
		out = append(out, chat(c))
	}
	httpx.JSON(w, 200, map[string]any{"chats": out})
}
func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	v, e := h.Service.Repo.History(r.Context(), owner(r).ID, r.PathValue("id"))
	if e != nil {
		fail(w, e)
		return
	}
	messages := []messageDTO{}
	for _, m := range v.Messages {
		messages = append(messages, messageDTO{m.ID, m.RunID, m.Role, m.Content, m.CreatedAt})
	}
	runs := []runDTO{}
	for _, v := range v.Runs {
		runs = append(runs, run(v))
	}
	actions := []actionDTO{}
	for _, v := range v.Actions {
		actions = append(actions, action(v))
	}
	cards := []cardDTO{}
	for _, c := range v.Cards {
		cards = append(cards, cardDTO{c.RunID, c.Kind, c.Data})
	}
	var latest *runDTO
	if len(runs) > 0 {
		latest = &runs[len(runs)-1]
	}
	httpx.JSON(w, 200, map[string]any{"chat": chat(v.Chat), "messages": messages, "runs": runs, "latest_run": latest, "actions": actions, "cards": cards})
}
