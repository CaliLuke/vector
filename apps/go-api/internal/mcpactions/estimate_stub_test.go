package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/projectconfiguration"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
)

type estimateApplicationStub struct{}

func (estimateApplicationStub) ReadEstimate(context.Context, projectconfiguration.EstimateRead) (json.RawMessage, error) {
	return nil, errors.New("unexpected estimate read")
}
func (estimateApplicationStub) MutateEstimate(context.Context, projectmutations.Command) (projectmutations.Result, error) {
	return projectmutations.Result{}, errors.New("unexpected estimate mutation")
}
