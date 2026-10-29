package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
)

func invitationCommandFields(c *InvitationCommand) (map[string]any, error) {
	input, err := djangoinput.Parse(c.Body, c.ContentType)
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
	// ChoiceField uses Python str(value), so a JSON floating token is not an
	// integer choice even when its mathematical value is integral.
	switch v := value.(type) {
	case json.Number:
		return invitationRole(string(v))
	case int:
		if v == 5 || v == 15 || v == 20 {
			return v, true
		}
	case string:
		if v == "5" || v == "15" || v == "20" {
			n, _ := strconv.Atoi(v)
			return n, true
		}
	}
	return 0, false
}

func invitationInt(value any) (int, error) {
	switch v := value.(type) {
	case json.Number:
		f, e := strconv.ParseFloat(string(v), 64)
		return int(f), e
	case float64:
		return int(v), nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case string:
		return strconv.Atoi(strings.TrimSpace(v))
	default:
		return 0, errors.New("invalid integer")
	}
}

func invitationTruth(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case json.Number:
		f, _ := strconv.ParseFloat(string(v), 64)
		return f != 0
	case float64:
		return v != 0
	case []any:
		return len(v) != 0
	case map[string]any:
		return len(v) != 0
	default:
		return true
	}
}

var workspaceInvitationFields = []string{"role", "accepted", "created_by", "updated_by", "deleted_at"}

func invitationChoiceMessages(value any) []string {
	if value == nil {
		return []string{"This field may not be null."}
	}
	label := fmt.Sprint(value)
	if flag, ok := value.(bool); ok {
		label = "False"
		if flag {
			label = "True"
		}
	}
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
				parsed, err := djangoinput.UUID(value)
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
				parsed, err := pythoncompat.DjangoDateTime(text, location)
				if !valid || err != nil {
					messages = []string{"Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."}
				} else {
					value = parsed
				}
			}
		case "company_role":
			if value != nil {
				value, messages = djangoinput.TextField(value, true, 0, nil)
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
		id, err := djangoinput.UUID(raw)
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
	var first, second *djangoapi.HTTPError
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
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		switch strings.ToLower(v) {
		case "true", "1", "yes", "t", "on":
			return true, true
		case "false", "0", "no", "f", "off":
			return false, true
		}
	case json.Number:
		number, err := strconv.ParseFloat(string(v), 64)
		if err == nil && (number == 0 || number == 1) {
			return number == 1, true
		}
	case float64:
		if v == 1 {
			return true, true
		}
		if v == 0 {
			return false, true
		}
	}
	return false, false
}

// Model BooleanField coercion differs from DRF BooleanField and from Python
// truthiness. Django leaves the instance attribute uncoerced after saving.
func invitationModelBoolean(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case json.Number:
		if v == "1" || v == "1.0" {
			return true, true
		}
		if v == "0" || v == "0.0" {
			return false, true
		}
	case string:
		switch v {
		case "t", "True", "1":
			return true, true
		case "f", "False", "0":
			return false, true
		}
	}
	return false, false
}
