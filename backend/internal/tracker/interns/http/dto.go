package internshttp

import (
	"ai-buddy/internal/tracker/interns"
)

type profileResponse struct {
	ID           string  `json:"authentik_id"`
	Name         string  `json:"name"`
	Username     string  `json:"username"`
	Email        string  `json:"email"`
	Role         string  `json:"role"`
	Team         string  `json:"team"`
	StartDate    *string `json:"start_date"`
	Goals        string  `json:"goals"`
	Expectations string  `json:"expectations"`
}

func fromProfile(model interns.Profile) profileResponse {
	return profileResponse{
		ID:           model.ID,
		Name:         model.Name,
		Username:     model.Username,
		Email:        model.Email,
		Role:         model.Role,
		Team:         model.Team,
		StartDate:    model.StartDate,
		Goals:        model.Goals,
		Expectations: model.Expectations,
	}
}

func fromProfiles(models []interns.Profile) []profileResponse {
	if models == nil {
		return nil
	}

	result := make([]profileResponse, len(models))
	for i, model := range models {
		result[i] = fromProfile(model)
	}

	return result
}

type internsResponse struct {
	Interns []profileResponse `json:"interns"`
}

type mentorResponse struct {
	Mentor *profileResponse `json:"mentor"`
}

func fromMentor(model *interns.Profile) mentorResponse {
	if model == nil {
		return mentorResponse{}
	}

	mentor := fromProfile(*model)
	return mentorResponse{Mentor: &mentor}
}
