package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/releases"
)

type releaseApplicationStub struct{}

func (releaseApplicationStub) Read(context.Context, releases.Query) (json.RawMessage, error) {
	return nil, errors.New("unexpected release read")
}
func (releaseApplicationStub) Mutate(context.Context, releases.Command) (projectmutations.Result, error) {
	return projectmutations.Result{}, errors.New("unexpected release mutation")
}
