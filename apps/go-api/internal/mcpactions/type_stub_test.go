package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
)

type typeApplicationStub struct{}

func (typeApplicationStub) ReadAPI(context.Context, workitemtypes.APIRead) (json.RawMessage, error) {
	return nil, errors.New("unexpected type read")
}
func (typeApplicationStub) MutateAPI(context.Context, workitemtypes.APIMutation) (projectmutations.Result, error) {
	return projectmutations.Result{}, errors.New("unexpected type mutation")
}
func (typeApplicationStub) APIVisibility(context.Context, workitemtypes.APIScope) (workitemtypes.APICapabilities, error) {
	return workitemtypes.APICapabilities{}, nil
}
