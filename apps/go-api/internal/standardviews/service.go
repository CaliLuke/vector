package standardviews

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"uuid"
)

// Service is the session-authenticated application boundary for standard
// views. Session validation happens in the generated HTTP adapter; this
// boundary receives the resulting user UUID and performs tenant checks before
// every repository operation.
type Service struct{ repository *Repository }

func NewService(repository *Repository) (*Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("standard views repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) authorizeWorkspace(ctx context.Context, user uuid.UUID, slug string) error {
	ok, err := s.repository.AuthorizedWorkspace(ctx, user, slug)
	if err != nil {
		return fmt.Errorf("authorize workspace: %w", err)
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) authorizeProject(ctx context.Context, user uuid.UUID, slug string, project uuid.UUID) error {
	ok, err := s.repository.AuthorizedProject(ctx, user, slug, project)
	if err != nil {
		return fmt.Errorf("authorize project: %w", err)
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) ListWorkspace(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	if err := s.authorizeWorkspace(ctx, user, slug); err != nil {
		return nil, err
	}
	return s.repository.ListWorkspace(ctx, user, slug)
}
func (s *Service) GetWorkspace(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID) (json.RawMessage, error) {
	if err := s.authorizeWorkspace(ctx, user, slug); err != nil {
		return nil, err
	}
	body, err := s.repository.GetWorkspace(ctx, user, slug, id)
	if err == nil {
		_ = s.repository.RecordRecentVisit(ctx, user, slug, nil, id)
	}
	return body, err
}
func (s *Service) CreateWorkspace(ctx context.Context, user uuid.UUID, slug string, in ViewInput) (json.RawMessage, error) {
	if err := s.authorizeWorkspace(ctx, user, slug); err != nil {
		return nil, err
	}
	if err := validateInput(in, true); err != nil {
		return nil, err
	}
	return s.repository.CreateWorkspace(ctx, user, slug, in)
}
func (s *Service) UpdateWorkspace(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID, in ViewInput) (json.RawMessage, error) {
	if err := s.authorizeWorkspace(ctx, user, slug); err != nil {
		return nil, err
	}
	if err := validateInput(in, false); err != nil {
		return nil, err
	}
	return s.repository.UpdateWorkspace(ctx, user, slug, id, in)
}
func (s *Service) PatchWorkspace(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID, in ViewInput) (json.RawMessage, error) {
	if err := s.authorizeWorkspace(ctx, user, slug); err != nil {
		return nil, err
	}
	if err := validateInput(in, false); err != nil {
		return nil, err
	}
	return s.repository.PatchWorkspace(ctx, user, slug, id, in)
}
func (s *Service) DeleteWorkspace(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID) error {
	if err := s.authorizeWorkspace(ctx, user, slug); err != nil {
		return err
	}
	return s.repository.DeleteWorkspace(ctx, user, slug, id)
}

func (s *Service) ListProject(ctx context.Context, user uuid.UUID, slug string, project uuid.UUID) (json.RawMessage, error) {
	if err := s.authorizeProject(ctx, user, slug, project); err != nil {
		return nil, err
	}
	return s.repository.ListProject(ctx, user, slug, project)
}
func (s *Service) GetProject(ctx context.Context, user uuid.UUID, slug string, project, id uuid.UUID) (json.RawMessage, error) {
	if err := s.authorizeProject(ctx, user, slug, project); err != nil {
		return nil, err
	}
	body, err := s.repository.GetProject(ctx, user, slug, project, id)
	if err == nil {
		_ = s.repository.RecordRecentVisit(ctx, user, slug, &project, id)
	}
	return body, err
}
func (s *Service) CreateProject(ctx context.Context, user uuid.UUID, slug string, project uuid.UUID, in ViewInput) (json.RawMessage, error) {
	if err := s.authorizeProject(ctx, user, slug, project); err != nil {
		return nil, err
	}
	if err := validateInput(in, true); err != nil {
		return nil, err
	}
	return s.repository.CreateProject(ctx, user, slug, project, in)
}
func (s *Service) UpdateProject(ctx context.Context, user uuid.UUID, slug string, project, id uuid.UUID, in ViewInput) (json.RawMessage, error) {
	if err := s.authorizeProject(ctx, user, slug, project); err != nil {
		return nil, err
	}
	if err := validateInput(in, false); err != nil {
		return nil, err
	}
	return s.repository.UpdateProject(ctx, user, slug, project, id, in)
}
func (s *Service) PatchProject(ctx context.Context, user uuid.UUID, slug string, project, id uuid.UUID, in ViewInput) (json.RawMessage, error) {
	if err := s.authorizeProject(ctx, user, slug, project); err != nil {
		return nil, err
	}
	if err := validateInput(in, false); err != nil {
		return nil, err
	}
	return s.repository.PatchProject(ctx, user, slug, project, id, in)
}
func (s *Service) DeleteProject(ctx context.Context, user uuid.UUID, slug string, project, id uuid.UUID) error {
	if err := s.authorizeProject(ctx, user, slug, project); err != nil {
		return err
	}
	return s.repository.DeleteProject(ctx, user, slug, project, id)
}

func validateInput(in ViewInput, create bool) error {
	if create && in.Name == nil {
		return errors.New("view name is required")
	}
	if in.Name != nil && len([]rune(*in.Name)) > 255 {
		return errors.New("view name is too long")
	}
	return nil
}
