package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	// Project packages.
	"backend/internal/buddy/agent"
	buddyhandler "backend/internal/buddy/handler"
	"backend/internal/ollama"
	"backend/internal/tracker/handler"
	"backend/internal/tracker/service"
	"backend/internal/users"
	"backend/internal/users/middleware"

	"github.com/DATA-DOG/go-sqlmock"
)

type flowUsers struct{}

func (flowUsers) Get(_ context.Context, id string) (users.User, error) {
	role := users.RoleIntern
	if strings.HasPrefix(id, "mentor") {
		role = users.RoleMentor
	}
	return users.User{ID: id, Role: role}, nil
}

type flowModel struct {
	tool string
}

func (m flowModel) Chat(
	_ context.Context, messages []ollama.Message, _ []ollama.Tool, _ func(string) error,
) (ollama.Response, error) {
	if messages[len(messages)-1].Role == "tool" {
		return ollama.Response{Message: ollama.Message{Content: "Подтвердите действие кнопкой"}}, nil
	}

	args := []byte(`{}`)
	switch m.tool {
	case "prepare_task_completion":
		args = []byte(`{"task_id":"saved"}`)
	case "prepare_task":
		args = []byte(`{"description":"Новая задача","deadline":"2030-01-01T00:00:00Z"}`)
	}

	return ollama.Response{Message: ollama.Message{
		Calls: []ollama.Call{{Function: ollama.Function{
			Name:      m.tool,
			Arguments: args,
		}}},
	}}, nil
}

func TestCardsAndExplicitPost(t *testing.T) {
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, tool := range []string{"list_tasks", "prepare_task_completion", "prepare_task"} {
		t.Run(tool, func(t *testing.T) {
			repo, mock := mockRepository(t)
			tasks := service.NewService(repo)
			a, err := agent.NewAgent(flowModel{tool: tool}, nil, "../../../prompts/buddy.md", nil, tasks)
			if err != nil {
				t.Fatal(err)
			}

			selected := middleware.Middleware(flowUsers{})
			mux := http.NewServeMux()
			h := handler.NewHandler(tasks)
			mux.Handle("POST /api/buddy/messages", selected(http.HandlerFunc(buddyhandler.NewHandler(a, 6).Send)))
			mux.Handle("GET /api/tracker/tasks", selected(http.HandlerFunc(h.List)))
			mux.Handle("GET /api/tracker/interns/{internID}/tasks", selected(http.HandlerFunc(h.ListIntern)))
			mux.Handle("POST /api/tracker/tasks", selected(http.HandlerFunc(h.Create)))
			mux.Handle("POST /api/tracker/tasks/{id}/complete", selected(http.HandlerFunc(h.Complete)))
			request := func(method, path, user, body string) *httptest.ResponseRecorder {
				r := httptest.NewRequest(method, path, strings.NewReader(body))
				r.AddCookie(&http.Cookie{Name: "demo_user", Value: user})
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, r)

				return w
			}

			expectList := func(user string, empty bool) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "description", "deadline", "status", "created_at"})
				if empty {
					rows.AddRow(nil, nil, nil, nil, "pending", nil)
				} else {
					rows.AddRow("saved", user, "Задача", now, "pending", now)
				}

				mock.ExpectQuery(regexp.QuoteMeta(listSQL)).WithArgs(user, "intern", user).
					WillReturnRows(rows).RowsWillBeClosed()
			}

			// SQL expectations allow only reads while preparing a card, including text consent.

			for _, user := range []string{"intern.a", "intern.b"} {
				for range 2 {
					if tool != "prepare_task" {
						expectList(user, false)
					}

					w := request("POST", "/api/buddy/messages", user,
						`{"messages":[{"role":"assistant",`+
							`"content":"Подтвердите действие"},{"role":"user","content":"Да, подтверждаю"}]}`)
					event := map[string]string{
						"list_tasks": "tasks", "prepare_task": "task", "prepare_task_completion": "task_completion",
					}[tool]
					if w.Code != 200 || !strings.Contains(w.Body.String(), "event: "+event+"\n") ||
						!strings.Contains(w.Body.String(), "event: done\n") {
						t.Fatalf("%d %s", w.Code, w.Body)
					}

					if tool != "prepare_task" && !strings.Contains(w.Body.String(), `"user_id":"`+user+`"`) {
						t.Fatal(w.Body)
					}
				}
			}

			if tool == "list_tasks" {
				for _, accessible := range []bool{false, true} {
					rows := sqlmock.NewRows([]string{"id", "user_id", "description", "deadline", "status", "created_at"})
					if accessible {
						rows.AddRow("saved", "intern.a", "Задача", now, "pending", now)
					}
					mock.ExpectQuery(regexp.QuoteMeta(listSQL)).WithArgs("intern.a", "mentor", "mentor.a").
						WillReturnRows(rows).RowsWillBeClosed()
					w := request("GET", "/api/tracker/interns/intern.a/tasks", "mentor.a", "")
					expected := 404
					if accessible {
						expected = 200
					}
					if w.Code != expected {
						t.Fatalf("mentor access %d %s", w.Code, w.Body)
					}
				}

				expectList("intern.a", true)
				w := request("POST", "/api/buddy/messages", "intern.a", `{"messages":[{"role":"user","content":"Мои задачи"}]}`)
				if !strings.Contains(w.Body.String(), "event: tasks\ndata: []\n\n") {
					t.Fatal(w.Body)
				}

				expectList("intern.a", false)
				w = request("GET", "/api/tracker/tasks", "intern.a", "")
				if w.Code != 200 || !strings.Contains(w.Body.String(), `"status":"pending"`) {
					t.Fatal(w.Body)
				}
			}

			if tool == "prepare_task_completion" {
				mock.ExpectQuery(regexp.QuoteMeta(completeSQL)).WithArgs("saved", "intern.b", "intern").
					WillReturnRows(
						sqlmock.NewRows([]string{"id", "user_id", "description", "deadline", "status", "created_at"}))
				if w := request("POST", "/api/tracker/tasks/saved/complete", "intern.b", ""); w.Code != 404 {
					t.Fatal(w.Code)
				}

				mock.ExpectQuery(regexp.QuoteMeta(completeSQL)).WithArgs("saved", "intern.a", "intern").
					WillReturnRows(
						sqlmock.NewRows([]string{"id", "user_id", "description", "deadline", "status", "created_at"}).
							AddRow("saved", "intern.a", "Задача", now, "completed", now))
				w := request("POST", "/api/tracker/tasks/saved/complete", "intern.a", "")
				if w.Code != 200 || !strings.Contains(w.Body.String(), `"status":"completed"`) {
					t.Fatalf("%d %s", w.Code, w.Body)
				}
			}

			if tool == "prepare_task" {
				mock.ExpectQuery(regexp.QuoteMeta(addSQL)).WithArgs(sqlmock.AnyArg(), "intern.a", "Окончательные поля", now).
					WillReturnRows(sqlmock.NewRows([]string{"created_at", "status"}).AddRow(now, "pending"))
				w := request("POST", "/api/tracker/tasks", "intern.a", `{"description":"Окончательные поля",`+
					`"deadline":"2030-01-01T00:00:00Z","user_id":"intern.b"}`)
				var body map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}

				if w.Code != 201 || body["user_id"] != "intern.a" || body["description"] != "Окончательные поля" {
					t.Fatalf("%d %s", w.Code, w.Body)
				}
			}
		})
	}
}
