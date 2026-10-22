package workspacesidebarpreferences

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"uuid"
)

type serviceRepo struct {
	authorized bool
	err        error
	called     bool
}

func (r *serviceRepo) Authorized(context.Context, uuid.UUID, string) (bool, error) {
	return r.authorized, r.err
}
func (r *serviceRepo) List(context.Context, uuid.UUID, string) (json.RawMessage, error) {
	r.called = true
	return json.RawMessage(`{}`), r.err
}
func (r *serviceRepo) Patch(context.Context, uuid.UUID, string, []Patch) (json.RawMessage, error) {
	r.called = true
	return json.RawMessage(`{"message":"Successfully updated"}`), r.err
}

func TestServiceMapsAuthorizationAndRepositoryFailures(t *testing.T) {
	for _, test := range []struct {
		name string
		repo *serviceRepo
		user uuid.UUID
		want error
	}{
		{name: "anonymous", repo: &serviceRepo{authorized: true}, want: ErrUnauthenticated},
		{name: "forbidden", repo: &serviceRepo{}, user: uuid.MustParse("00000000-0000-4000-8000-000000000001"), want: ErrForbidden},
		{name: "authorization database failure", repo: &serviceRepo{err: errors.New("database")}, user: uuid.MustParse("00000000-0000-4000-8000-000000000001"), want: &InternalError{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, err := NewService(test.repo)
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.Get(context.Background(), test.user, "demo")
			if !errors.Is(err, test.want) && (test.name != "authorization database failure" || !errors.As(err, new(*InternalError))) {
				t.Fatalf("Get() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestNewServiceRejectsNilRepository(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatal("NewService(nil) returned nil error")
	}
}
