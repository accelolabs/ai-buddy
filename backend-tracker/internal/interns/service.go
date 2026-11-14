package interns

import "context"

type Repository interface {
	Intern(context.Context, string) (Profile, error)
	Mentor(context.Context, string) (*Profile, error)
	Interns(context.Context, string) ([]Profile, error)
}
type Service struct{ repository Repository }

func New(repository Repository) *Service { return &Service{repository: repository} }
func (s *Service) Intern(ctx context.Context, id string) (Profile, error) {
	return s.repository.Intern(ctx, id)
}
func (s *Service) Mentor(ctx context.Context, id string) (*Profile, error) {
	if _, err := s.repository.Intern(ctx, id); err != nil {
		return nil, err
	}
	return s.repository.Mentor(ctx, id)
}
func (s *Service) Interns(ctx context.Context, mentorID string) ([]Profile, error) {
	return s.repository.Interns(ctx, mentorID)
}
