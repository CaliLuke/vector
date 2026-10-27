package systemapi

import (
	"context"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"net/http"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	store "github.com/CaliLuke/go-plane/internal/milestones"
	api "github.com/CaliLuke/go-plane/milestones/gen/milestones"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

// MilestoneRequestDecoder keeps the shared body limits, media handling and
// non-null body checks while enforcing the native contract's closed fields.
func MilestoneRequestDecoder(r *http.Request) loomhttp.Decoder {
	return loomhttp.EncodingFunc(func(value any) error {
		var raw jsontext.Value
		if err := loomhttp.RequestDecoder(r).Decode(&raw); err != nil {
			return err
		}
		return jsonv2.Unmarshal(raw, value, loom.JSONOptions(), jsonv2.RejectUnknownMembers(true))
	})
}
func milestoneProjection[T any](value any) (*T, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, milestoneError(err)
	}
	projected := new(T)
	if err = json.Unmarshal(raw, projected); err != nil {
		return nil, milestoneError(err)
	}
	return projected, nil
}
func milestoneQuery(ctx context.Context, cursor, perPage *string) djangoapi.Query {
	query := djangoapi.Query{Params: map[string]string{}, Present: map[string]bool{}}
	for key, value := range map[string]*string{"cursor": cursor, "per_page": perPage} {
		if value != nil {
			query.Params[key] = *value
			query.Present[key] = true
		}
	}
	for _, key := range []string{"cursor", "per_page"} {
		if original := ReadServiceQueryValues(ctx)[key]; len(original) > 0 {
			query.Params[key], query.Present[key] = original[len(original)-1], true
		}
	}
	return query
}
func milestoneFields(title, date, source, id loom.Nullable[string]) store.Fields {
	return store.Fields{Title: milestoneField(title), TargetDate: milestoneField(date), ExternalSource: milestoneField(source), ExternalID: milestoneField(id)}
}
func milestoneField(value loom.Nullable[string]) store.StringField {
	field := store.StringField{Present: value.Present()}
	if text, ok := value.Value(); ok {
		field.Value = &text
	}
	return field
}
func milestoneWriteResult(value store.MutationResult, extended bool, err error) (*api.MilestoneWriteResult, error) {
	if err != nil {
		return nil, milestoneError(err)
	}
	if extended {
		receipt, projectErr := milestoneProjection[api.MilestoneMutationReceipt](value)
		if projectErr != nil {
			return nil, projectErr
		}
		result := api.NewMilestoneWriteResultMilestoneMutationReceipt(receipt)
		return &result, nil
	}
	record, err := milestoneProjection[api.MilestoneRecord](value.Result)
	if err != nil {
		return nil, err
	}
	result := api.NewMilestoneWriteResultMilestoneRecord(record)
	return &result, nil
}
func milestoneEmptyResult(value store.MutationResult, extended bool, err error) (*api.MilestoneEmptyResult, error) {
	if err != nil {
		return nil, milestoneError(err)
	}
	result := &api.MilestoneEmptyResult{}
	if extended {
		receipt, projectErr := milestoneProjection[api.MilestoneMutationReceipt](value)
		if projectErr != nil {
			return nil, projectErr
		}
		outcome := "extended"
		result.Outcome, result.Receipt = &outcome, receipt
	}
	return result, nil
}
