package testutil

import (
	"encoding/json"
	"reflect"
	"testing"
)

// JSON compares complete contracts, including extra fields, nulls and empty lists.
func JSON(t *testing.T, response any, want string) {
	t.Helper()
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(raw, &gotValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("got %s\nwant %s", raw, want)
	}
}
