package mcpactions_test

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
)

type resourceArchiveStub struct{}

func (resourceArchiveStub) MutateResource(context.Context, projectmutations.Command) (projectmutations.Result, error) {
	return projectmutations.Result{}, errors.New("unexpected archive mutation")
}
