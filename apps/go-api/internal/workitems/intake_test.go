package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/go-plane/internal/requestinput"
)

func TestIntakeStatusUsesChoiceSpellingInsteadOfIntegerCoercion(t *testing.T) {
	for _, raw := range []any{"1.0", json.Number("1.0"), "+1", "01", true} {
		_, err := (&Repository{}).validateIntake(context.Background(), Request{}, nil, requestinput.Input{Fields: map[string]any{"status": raw}})
		var response *apicontract.HTTPError
		if !errors.As(err, &response) || response.Status != 400 || response.Body["status"] == nil {
			t.Fatalf("status=%#v error=%v", raw, err)
		}
	}
	for _, raw := range []any{"0", json.Number("0")} {
		values, err := (&Repository{}).validateIntake(context.Background(), Request{}, nil, requestinput.Input{Fields: map[string]any{"status": raw}})
		if err != nil || values["status"] != int64(0) {
			t.Fatalf("status=%#v values=%#v error=%v", raw, values, err)
		}
	}
}
