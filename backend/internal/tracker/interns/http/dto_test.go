package internshttp

import (
	"ai-buddy/internal/tracker/dto"
	"testing"

	"ai-buddy/internal/testutil"
	"ai-buddy/internal/tracker/interns"
)

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
	testutil.JSON(t, dto.FromProfile(profile), profileJSON)
	testutil.JSON(t, internsResponse{Interns: dto.Profiles([]interns.Profile{profile})}, `{"interns":[`+profileJSON+`]}`)
	testutil.JSON(t, fromMentor(&profile), `{"mentor":`+profileJSON+`}`)
	testutil.JSON(t, fromMentor(nil), `{"mentor":null}`)
	testutil.JSON(t, internsResponse{Interns: dto.Profiles(nil)}, `{"interns":null}`)
	testutil.JSON(t, internsResponse{Interns: dto.Profiles([]interns.Profile{})}, `{"interns":[]}`)
	profile.StartDate = nil
	testutil.JSON(t, dto.FromProfile(profile), `{"authentik_id":"intern","name":"Name","username":"username","email":"user@example.com","role":"intern","team":"backend","start_date":null,"goals":"goals","expectations":"expectations"}`)
}
