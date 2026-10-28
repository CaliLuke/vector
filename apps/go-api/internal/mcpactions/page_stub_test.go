package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/pagereads"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
)

type pageApplicationStub struct{}

func (pageApplicationStub) ReadPageAPI(context.Context, pagereads.PageAPIQuery) (json.RawMessage, error) {
	return nil, errors.New("unexpected page read")
}
func (pageApplicationStub) MutatePageAPI(context.Context, pagereads.PageAPICommand) (projectmutations.Result, error) {
	return projectmutations.Result{}, errors.New("unexpected page mutation")
}
