package workspacesidebarpreferences

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"
)

var ErrUnauthenticated = errors.New("workspace sidebar preferences authentication required")
var ErrForbidden = errors.New("workspace sidebar preferences access forbidden")

// InternalError preserves the distinction between an unavailable dependency
// and an authorization denial for the HTTP adapter.
type InternalError struct{ Err error }

func (e *InternalError) Error() string {
	if e == nil || e.Err == nil {
		return "workspace sidebar preferences internal error"
	}
	return e.Err.Error()
}

func (e *InternalError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type serviceRepository interface {
	Authorized(context.Context, uuid.UUID, string) (bool, error)
	List(context.Context, uuid.UUID, string) (json.RawMessage, error)
	Patch(context.Context, uuid.UUID, string, []Patch) (json.RawMessage, error)
}

// Service owns authorization and repository error translation. The generated
// HTTP adapter can decode the raw JSON into Loom's generated result types.
type Service struct{ repository serviceRepository }

func NewService(repository serviceRepository) (*Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("workspace sidebar preferences repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) authorize(ctx context.Context, user uuid.UUID, slug string) error {
	ok, err := s.repository.Authorized(ctx, user, slug)
	if err != nil {
		return &InternalError{Err: err}
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) Get(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	if user == uuid.Nil() {
		return nil, ErrUnauthenticated
	}
	if err := s.authorize(ctx, user, slug); err != nil {
		return nil, err
	}
	body, err := s.repository.List(ctx, user, slug)
	if err != nil {
		return nil, &InternalError{Err: err}
	}
	return body, nil
}

func (s *Service) Patch(ctx context.Context, user uuid.UUID, slug string, patches []Patch) (json.RawMessage, error) {
	if user == uuid.Nil() {
		return nil, ErrUnauthenticated
	}
	if err := s.authorize(ctx, user, slug); err != nil {
		return nil, err
	}
	body, err := s.repository.Patch(ctx, user, slug, patches)
	if err != nil {
		return nil, &InternalError{Err: err}
	}
	return body, nil
}

var _ serviceRepository = (*Repository)(nil)
