package backgroundtasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/CaliLuke/selvedge"
)

func admitOrReconcile[I, O any](
	ctx context.Context,
	app *selvedge.App,
	workflow *selvedge.WorkflowFunc[I, O],
	instanceID string,
	input I,
) (string, error) {
	startedID, startErr := selvedge.StartWorkflow(ctx, app, workflow, input, selvedge.WithInstanceID(instanceID))
	if startErr == nil {
		return startedID, nil
	}
	instance, lookupErr := app.GetInstance(ctx, instanceID)
	if lookupErr != nil {
		return "", fmt.Errorf("start workflow %s: %w (reconcile existing instance: %v)", instanceID, startErr, lookupErr)
	}
	expectedInput, marshalErr := json.Marshal(input)
	if marshalErr != nil {
		return "", fmt.Errorf("encode expected workflow input for %s: %w", instanceID, marshalErr)
	}
	if instance.WorkflowName != workflow.Name() || !bytes.Equal(instance.InputData, expectedInput) {
		return "", fmt.Errorf("workflow ID %s already belongs to workflow %q with different input; original admission error: %w", instanceID, instance.WorkflowName, startErr)
	}
	return instanceID, nil
}
