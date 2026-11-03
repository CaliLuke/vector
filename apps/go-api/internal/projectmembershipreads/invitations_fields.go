package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CaliLuke/go-plane/internal/datetime"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/go-plane/internal/requestinput"
)

func invitationCommandFields(c *InvitationCommand) (map[string]any, error) {
	input, err := requestinput.Parse(c.Body, c.ContentType)
	if err != nil {
		return nil, err
	}
	if input.Kind != "" {
		return nil, errors.New("invitation data must be an object")
	}
	c.rawBody = input.Raw
	return input.Fields, nil
}

func invitationRole(value any) (int, bool) {
	role, err := memberRole(value)
	return role, err == nil
}

func invitationInt(value any) (int, error) {
	number, err := requestinput.IntegerField(value)
	if err != nil || !number.IsInt64() {
		return 0, inviteFailure(400, "Invalid integer")
	}
	return int(number.Int64()), nil
}

func invitationTruth(value any) bool { return requestinput.Nonempty(value) }

var workspaceInvitationFields = []string{"role", "accepted", "created_by", "updated_by", "deleted_at"}

func invitationChoiceMessages(value any) []string {
	if value == nil {
		return []string{"This field may not be null."}
	}
	label := fmt.Sprint(value)
	return []string{fmt.Sprintf("\"%s\" is not a valid choice.", label)}
}

func invitationFieldUpdates(fields map[string]any, id, actor, zone string, keys []string) ([]string, []any, error) {
	updates := []string{"updated_at=clock_timestamp()", "updated_by_id=$2"}
	args := []any{id, actor}
	failures := map[string]any{}
	for _, key := range keys {
		value, present := fields[key]
		if !present {
			continue
		}
		column := key
		var messages []string
		nullable := key == "company_role" || key == "created_by" || key == "updated_by" || key == "deleted_at"
		if value == nil && !nullable {
			failures[key] = []string{"This field may not be null."}
			continue
		}
		switch key {
		case "role":
			n, valid := invitationRole(value)
			if !valid {
				messages = invitationChoiceMessages(value)
			} else {
				value = n
			}
		case "is_active", "accepted":
			n, valid := invitationBoolean(value)
			if !valid {
				messages = []string{"Must be a valid boolean."}
			} else {
				value = n
			}
		case "workspace", "created_by", "updated_by":
			column += "_id"
			if value != nil {
				parsed, err := requestinput.UUID(value)
				if err != nil {
					messages = []string{fmt.Sprintf("“%v” is not a valid UUID.", value)}
				} else {
					value = parsed.String()
				}
			}
		case "deleted_at":
			if value != nil {
				text, valid := value.(string)
				location := time.UTC
				if zone != "" {
					if loaded, err := time.LoadLocation(zone); err == nil {
						location = loaded
					}
				}
				parsed, err := datetime.Parse(text, location)
				if !valid || err != nil {
					messages = []string{"Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."}
				} else {
					value = parsed
				}
			}
		case "company_role":
			if value != nil {
				value, messages = requestinput.TextField(value, true, 0)
			}
		default:
			data, err := json.Marshal(value)
			if err != nil {
				return nil, nil, err
			}
			value = string(data)
		}
		if len(messages) > 0 {
			failures[key] = messages
			continue
		}
		// AuditModel.save validates submitted updated_by through the serializer,
		// then unconditionally replaces it with the authenticated request actor.
		if key == "updated_by" {
			continue
		}
		args = append(args, value)
		updates = append(updates, fmt.Sprintf("%s=$%d", column, len(args)))
	}
	if len(failures) > 0 {
		return nil, nil, invitationHTTP(400, failures)
	}
	return updates, args, nil
}

func (r *InvitationsRepository) invitationRelationErrors(ctx context.Context, fields map[string]any, keys []string) error {
	failures := map[string]any{}
	for _, key := range keys {
		if key != "workspace" && key != "created_by" && key != "updated_by" {
			continue
		}
		raw, present := fields[key]
		if !present || raw == nil {
			continue
		}
		id, err := requestinput.UUID(raw)
		if err != nil {
			continue
		}
		table, predicate := "users", ""
		if key == "workspace" {
			table = "workspaces"
			predicate = " AND deleted_at IS NULL"
		}
		var exists bool
		if err = r.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id=$1`+predicate+`)`, id.String()).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			failures[key] = []string{fmt.Sprintf("Invalid pk \"%v\" - object does not exist.", raw)}
		}
	}
	if len(failures) > 0 {
		return invitationHTTP(400, failures)
	}
	return nil
}

func invitationValidationErrors(left, right error) error {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	var first, second *apicontract.HTTPError
	if errors.As(left, &first) && errors.As(right, &second) && first.Status == 400 && second.Status == 400 {
		body := map[string]any{}
		for key, value := range first.Body {
			body[key] = value
		}
		for key, value := range second.Body {
			body[key] = value
		}
		return invitationHTTP(400, body)
	}
	return right
}

func invitationBoolean(value any) (bool, bool) {
	boolean, message := requestinput.BooleanField(value)
	return boolean, message == ""
}
