package mcpactions_test

import (
	"context"
	"errors"

	labels "github.com/CaliLuke/go-plane/internal/projectconfiguration"
	"github.com/google/uuid"
)

type labelWriteStub struct{ visible bool }

func (s *labelWriteStub) VisibleLabelWrites(context.Context, uuid.UUID, uuid.UUID, string) (bool, error) {
	return s.visible, nil
}
func (s *labelWriteStub) MutateLabel(context.Context, labels.LabelMutation) (labels.LabelMutationResult, error) {
	return labels.LabelMutationResult{}, errors.New("unexpected label write")
}
