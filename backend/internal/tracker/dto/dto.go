package dto

import (
	"time"

	"ai-buddy/internal/tracker/interns"
	"ai-buddy/internal/tracker/tasks"
)

type Task struct {
	ID          string     `json:"id"`
	InternID    string     `json:"intern_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Deadline    string     `json:"deadline"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Status      string     `json:"status"`
	Overdue     bool       `json:"overdue"`
}

func FromTask(model tasks.Task) Task {
	return Task{
		ID:          model.ID,
		InternID:    model.InternID,
		Title:       model.Title,
		Description: model.Description,
		Deadline:    model.Deadline,
		CreatedAt:   model.CreatedAt,
		CompletedAt: model.CompletedAt,
		Status:      model.Status,
		Overdue:     model.Overdue,
	}
}

func Tasks(models []tasks.Task) []Task {
	if models == nil {
		return nil
	}

	result := make([]Task, len(models))
	for i, model := range models {
		result[i] = FromTask(model)
	}

	return result
}

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

func FromProfile(model interns.Profile) Profile {
	return Profile{
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

func Profiles(models []interns.Profile) []Profile {
	if models == nil {
		return nil
	}

	result := make([]Profile, len(models))
	for i, model := range models {
		result[i] = FromProfile(model)
	}

	return result
}
