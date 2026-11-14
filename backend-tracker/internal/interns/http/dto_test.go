package internshttp

import (
	"encoding/json"
	"reflect"
	"testing"

	"ai-buddy/backend-tracker/internal/interns"
)

func checkJSON(t *testing.T, response any, want string) {
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

func TestResponseContracts(t *testing.T) {
	start := "2026-11-01"
	profile := interns.Profile{
		ID:           "intern",
		Name:         "Name",
		Username:     "username",
		Email:        "user@example.com",
		Role:         "intern",
		Team:         "backend",
		StartDate:    &start,
		Goals:        "goals",
		Expectations: "expectations",
	}
	const profileJSON = `{"authentik_id":"intern","name":"Name","username":"username","email":"user@example.com","role":"intern","team":"backend","start_date":"2026-11-01","goals":"goals","expectations":"expectations"}`
	checkJSON(t, fromProfile(profile), profileJSON)
	checkJSON(t, internsResponse{Interns: fromProfiles([]interns.Profile{profile})}, `{"interns":[`+profileJSON+`]}`)
	checkJSON(t, fromMentor(&profile), `{"mentor":`+profileJSON+`}`)
	checkJSON(t, fromMentor(nil), `{"mentor":null}`)
	checkJSON(t, internsResponse{Interns: fromProfiles(nil)}, `{"interns":null}`)
	checkJSON(t, internsResponse{Interns: fromProfiles([]interns.Profile{})}, `{"interns":[]}`)
	profile.StartDate = nil
	checkJSON(t, fromProfile(profile), `{"authentik_id":"intern","name":"Name","username":"username","email":"user@example.com","role":"intern","team":"backend","start_date":null,"goals":"goals","expectations":"expectations"}`)
}
