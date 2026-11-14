package interns

type Profile struct {
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
