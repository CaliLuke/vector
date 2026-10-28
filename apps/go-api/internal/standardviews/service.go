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

func (s *Service) CreateWorkspace(ctx context.Context, user uuid.UUID, slug string, in ViewInput) (json.RawMessage, error) {
	if err := s.authorizeWorkspace(ctx, user, slug); err != nil {
		return nil, err
	}
	if err := validateCreateInput(in); err != nil {
		return nil, err
	}
	return s.repository.CreateWorkspace(ctx, user, slug, in)
}

func (s *Service) CreateProject(ctx context.Context, user uuid.UUID, slug string, project uuid.UUID, in ViewInput) (json.RawMessage, error) {
	if err := s.authorizeProject(ctx, user, slug, project); err != nil {
		return nil, err
	}
	if err := validateCreateInput(in); err != nil {
		return nil, err
	}
	return s.repository.CreateProject(ctx, user, slug, project, in)
}

func validateCreateInput(in ViewInput) error {
	if in.Name == nil {
		return errors.New("view name is required")
	}
	if in.Name != nil && len([]rune(*in.Name)) > 255 {
		return errors.New("view name is too long")
	}
	return nil
}
