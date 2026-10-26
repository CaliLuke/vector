package projectlabelreads

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/labelhierarchy"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type labelInput struct {
	fields     map[string]any
	surrogates map[string][]string
	kind       string
	null, form bool
}

func parseLabelInput(body io.Reader, contentType string) (labelInput, error) {
	input, err := djangoinput.Parse(body, contentType)
	var failure *djangoapi.HTTPError
	if errors.As(err, &failure) {
		err = &HTTPError{Status: failure.Status, Body: failure.Body}
	}
	return labelInput{input.Fields, input.Surrogates, input.Kind, input.Null, input.Form}, err
}
func pythonScalarString(value any) string { return djangoinput.ScalarString(value) }
func charField(value any, allowBlank bool, surrogates []string) (string, []string) {
	return djangoinput.CharField(value, allowBlank, surrogates)
}
func labelParentUUID(value any) (uuid.UUID, error) { return djangoinput.UUID(value) }
func labelFloatField(value any) (float64, string)  { return djangoinput.FloatField(value) }
func (r *WriteRepository) validateInput(ctx context.Context, tx pgx.Tx, input labelInput, label *Label, project, method string) error {
	if input.null {
		return &HTTPError{400, map[string]any{"non_field_errors": []string{"No data provided"}}}
	}
	if input.kind != "" {
		return &HTTPError{400, map[string]any{"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + input.kind + "."}}}
	}
	failures := map[string]any{}
	if parent, exists := input.fields["parent"]; exists {
		if parent == nil || parent == "" {
			label.Parent = nil
		} else {
			text := pythonScalarString(parent)
			parsed, err := labelParentUUID(parent)
			if _, boolean := parent.(bool); boolean {
				failures["parent"] = []string{"Incorrect type. Expected pk value, received bool."}
			} else if err != nil {
				failures["parent"] = []string{"“" + text + "” is not a valid UUID."}
			} else {
				id := parsed.String()
				valid, err := labelhierarchy.ValidateParent(ctx, tx, label.ID, id, project, label.WorkspaceID)
				if err != nil {
					return err
				}
				if !valid {
					failures["parent"] = []string{fmt.Sprintf("Invalid pk %q - object does not exist.", text)}
				} else {
					label.Parent = &id
				}
			}
		}
	}
	if name, present := input.fields["name"]; present {
		value, errs := charField(name, false, input.surrogates["name"])
		if len(errs) == 0 {
			duplicate, err := r.duplicate(ctx, nil, value, label.ID, false)
			if err != nil {
				return err
			}
			if duplicate {
				errs = append(errs, "Label with this name already exists.")
			}
		}
		if len(errs) == 0 {
			var projectContext *string
			if method != http.MethodPut {
				projectContext = &project
			}
			duplicate, err := r.duplicate(ctx, projectContext, value, label.ID, true)
			if err != nil {
				return err
			}
			if duplicate {
				errs = append(errs, "LABEL_NAME_ALREADY_EXISTS")
			}
		}
		if len(errs) > 0 {
			failures["name"] = errs
		} else {
			label.Name = value
		}
	} else if method != http.MethodPatch {
		failures["name"] = []string{"This field is required."}
	}
	if color, present := input.fields["color"]; present {
		value, errs := charField(color, true, input.surrogates["color"])
		if len(errs) > 0 {
			failures["color"] = errs
		} else {
			label.Color = value
		}
	}
	if order, present := input.fields["sort_order"]; present && (!input.form || order != "") {
		if order == nil {
			failures["sort_order"] = []string{"This field may not be null."}
		} else {
			value, failure := labelFloatField(order)
			if failure != "" {
				failures["sort_order"] = []string{failure}
			} else {
				label.SortOrder = value
			}
		}
	}
	if len(failures) > 0 {
		return &HTTPError{400, failures}
	}
	return nil
}
