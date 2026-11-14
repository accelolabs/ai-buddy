package trackerhttp

import (
	"ai-buddy/internal/tracker/interns"
)

type internsResponse struct {
	Interns []interns.Profile `json:"interns"`
}
