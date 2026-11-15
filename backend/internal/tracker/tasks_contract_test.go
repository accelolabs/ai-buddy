package tracker_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"ai-buddy/internal/tracker"
	"ai-buddy/internal/users"
)

func TestWireContract(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  string
	}{
		{tracker.Task{}, `{"id":"","intern_id":"","title":"","description":"","deadline":"","created_at":"0001-01-01T00:00:00Z","completed_at":null,"status":"","overdue":false}`},
		{users.User{}, `{"id":"","name":"","username":"","email":"","role":"","team":"","start_date":null,"goals":"","expectations":""}`},
		{[]tracker.Task(nil), `null`}, {[]tracker.Task{}, `[]`},
		{[]users.User(nil), `null`}, {[]users.User{}, `[]`},
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
