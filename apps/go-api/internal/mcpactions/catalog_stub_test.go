package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/customers"
	"github.com/CaliLuke/go-plane/internal/features"
	"github.com/CaliLuke/go-plane/internal/initiatives"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/templates"
)

type templateApplicationStub struct{}

func (templateApplicationStub) List(context.Context, templates.Query) (json.RawMessage, error) {
	return nil, errors.New("unexpected template read")
}
func (templateApplicationStub) Mutate(context.Context, templates.Command) (projectmutations.Result, error) {
	return projectmutations.Result{}, errors.New("unexpected template mutation")
}

type featureApplicationStub struct{}

func (featureApplicationStub) WorkspaceVisibility(context.Context, features.Query) (features.Capabilities, error) {
	return features.Capabilities{}, nil
}

func (featureApplicationStub) Read(context.Context, features.Query) (json.RawMessage, error) {
	return json.RawMessage(`{"initiatives":false}`), nil
}
func (featureApplicationStub) Mutate(context.Context, features.Command) (projectmutations.Result, error) {
	return projectmutations.Result{}, errors.New("unexpected feature mutation")
}

type initiativeApplicationStub struct{}

func (initiativeApplicationStub) Read(context.Context, initiatives.Query) (json.RawMessage, error) {
	return nil, errors.New("unexpected initiative read")
}
func (initiativeApplicationStub) Mutate(context.Context, initiatives.Command) (projectmutations.Result, error) {
	return projectmutations.Result{}, errors.New("unexpected initiative mutation")
}

type customerApplicationStub struct{}

func (customerApplicationStub) Read(context.Context, customers.Query) (json.RawMessage, error) {
	return nil, errors.New("unexpected customer read")
}
func (customerApplicationStub) Mutate(context.Context, customers.Command) (projectmutations.Result, error) {
	return projectmutations.Result{}, errors.New("unexpected customer mutation")
}
