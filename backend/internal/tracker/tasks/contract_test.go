package tasks_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"ai-buddy/internal/tracker/interns"
	"ai-buddy/internal/tracker/tasks"
)

func TestWireContract(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  string
	}{
		{tasks.Task{}, `{"id":"","intern_id":"","title":"","description":"","deadline":"","created_at":"0001-01-01T00:00:00Z","completed_at":null,"status":"","overdue":false}`},
		{interns.Profile{}, `{"authentik_id":"","name":"","username":"","email":"","role":"","team":"","start_date":null,"goals":"","expectations":""}`},
		{[]tasks.Task(nil), `null`}, {[]tasks.Task{}, `[]`},
		{[]interns.Profile(nil), `null`}, {[]interns.Profile{}, `[]`},
	} {
		got, err := json.Marshal(tc.value)
		var a, b any
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(got, &a); err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal([]byte(tc.want), &b); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("got %s; want %s", got, tc.want)
		}
	}
}
