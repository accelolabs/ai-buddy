package internshttp

import (
	"ai-buddy/internal/tracker/dto"
	"ai-buddy/internal/tracker/interns"
)

type internsResponse struct {
	Interns []dto.Profile `json:"interns"`
}

type mentorResponse struct {
	Mentor *dto.Profile `json:"mentor"`
}

func fromMentor(model *interns.Profile) mentorResponse {
	if model == nil {
		return mentorResponse{}
	}

	mentor := dto.FromProfile(*model)
	return mentorResponse{Mentor: &mentor}
}
