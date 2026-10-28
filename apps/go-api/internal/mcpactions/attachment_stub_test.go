package mcpactions_test

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
)

type attachmentApplicationStub struct{}

func (attachmentApplicationStub) Read(context.Context, assets.AttachmentQuery) (any, error) {
	return nil, errors.New("unexpected attachment read")
}
func (attachmentApplicationStub) MutateResource(context.Context, projectmutations.Command) (projectmutations.Result, error) {
	return projectmutations.Result{}, errors.New("unexpected attachment mutation")
}
