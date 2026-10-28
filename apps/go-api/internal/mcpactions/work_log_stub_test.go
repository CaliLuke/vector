package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/worklogs"
)

type workLogApplicationStub struct{}

func (workLogApplicationStub) List(context.Context, worklogs.Query) (json.RawMessage, error) {
	return nil, errors.New("unexpected work log list")
}
func (workLogApplicationStub) MutateResource(context.Context, projectmutations.Command) (projectmutations.Result, error) {
	return projectmutations.Result{}, errors.New("unexpected work log mutation")
}

func (workLogApplicationStub) Summary(context.Context, worklogs.Query) (json.RawMessage, error) {
	return nil, errors.New("unexpected project work log summary")
}
