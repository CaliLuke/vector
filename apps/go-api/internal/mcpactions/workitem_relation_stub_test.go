package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitemrelations"
)

type relationApplicationStub struct{}

func (relationApplicationStub) Read(context.Context, workitemrelations.Query) (json.RawMessage, error) {
	return nil, errors.New("unexpected relation read")
}
func (relationApplicationStub) Mutate(context.Context, workitemrelations.Command) (projectmutations.Result, error) {
	return projectmutations.Result{}, errors.New("unexpected relation mutation")
}
