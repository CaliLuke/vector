package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
)

type propertyApplicationStub struct{}

func (propertyApplicationStub) ReadPropertyAPI(context.Context, workitemtypes.PropertyAPIQuery) (json.RawMessage, error) {
	return nil, errors.New("unexpected property read")
}
func (propertyApplicationStub) MutatePropertyAPI(context.Context, workitemtypes.PropertyAPIMutation) (projectmutations.Result, error) {
	return projectmutations.Result{}, errors.New("unexpected property mutation")
}

func (propertyApplicationStub) ReadPropertyValueAPI(context.Context, workitemtypes.PropertyValueAPIQuery) (json.RawMessage, error) {
	return nil, errors.New("unexpected property value read")
}
