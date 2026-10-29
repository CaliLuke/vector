package pagereads

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/jackc/pgx/v5"
)

type pageCreateFields struct {
	name, color          string
	access               int
	labels               lifecycleLabelList
	parent               any
	viewProps, logoProps []byte
	locked               bool
	archivedAt           any
}

// pageCreateContent retains the raw create-context values and their database
// representations. The metadata and description-update serializers differ.
type pageCreateContent struct {
	html, json, binary, stripped any
}

func (repository *Repository) createPage(ctx context.Context, db lifecycleDatabase, user uuid.UUID, slug string, project uuid.UUID, timezone string, input djangoinput.Input) (int, json.RawMessage, error) {
	fields := input.Fields
	var workspace uuid.UUID
	var role int
	if err := repository.database.QueryRow(ctx, `SELECT w.id, pm.role FROM workspaces w JOIN projects pr ON pr.workspace_id=w.id AND pr.id=$2 AND pr.deleted_at IS NULL JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.member_id=$1 AND wm.is_active AND wm.deleted_at IS NULL JOIN project_members pm ON pm.project_id=pr.id AND pm.member_id=$1 AND pm.is_active AND pm.deleted_at IS NULL WHERE w.slug=$3 AND w.deleted_at IS NULL`, user, project, slug).Scan(&workspace, &role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil, lifecycleError(http.StatusForbidden, "You do not have permission to perform this action.")
		}
		return 0, nil, err
	}
	if role < 15 {
		return 0, nil, lifecycleError(http.StatusForbidden, "You do not have permission to perform this action.")
	}
	// PageViewSet.create performs this raw ORM lookup before constructing the
	// serializer. A truthy invalid or out-of-scope parent therefore takes
	// precedence over every serializer field error, including labels.
	var handlerParent *uuid.UUID
	if value, present := fields["parent"]; present && pythoncompat.Truthy(value) {
		parentID, err := lifecycleParentLookupUUID(value)
		if err != nil {
			return 0, nil, lifecycleError(http.StatusBadRequest, "Please provide valid detail")
		}
		exists, err := repository.lifecycleParentExists(ctx, parentID, workspace, project)
		if err != nil {
			return 0, nil, err
		}
		if !exists {
			return 0, nil, lifecycleError(http.StatusNotFound, "The required object does not exist.")
		}
		handlerParent = &parentID
	}
	values, err := repository.validatePageCreateFields(ctx, workspace, project, input, handlerParent)
	if err != nil {
		return 0, nil, err
	}
	description, err := parsePageCreateContent(fields)
	if err != nil {
		return 0, nil, err
	}
	page := uuid.New()
	if err := repository.writePage(ctx, db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO pages (id, workspace_id, name, description_json, description_binary, description_html, description_stripped, owned_by_id, access, color, parent_id, view_props, logo_props, is_global, sort_order, is_locked, archived_at, created_by_id, updated_by_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,false,65535,$14,$15,$8,NULL,now(),now())`, page, workspace, values.name, description.json, description.binary, description.html, description.stripped, user, values.access, values.color, values.parent, values.viewProps, values.logoProps, values.locked, values.archivedAt)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO project_pages (id, workspace_id, project_id, page_id, created_by_id, updated_by_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,NULL,now(),now())`, uuid.New(), workspace, project, page, user); err != nil {
			return err
		}
		for _, label := range values.labels.references {
			if _, err := tx.Exec(ctx, `INSERT INTO page_labels (id, workspace_id, page_id, label_id, created_by_id, updated_by_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,NULL,now(),now())`, uuid.New(), workspace, page, label.id, user); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return 0, nil, pageLifecycleDatabaseError(err)
	}
	if repository.publisher != nil || repository.nativeTasks != nil {
		taskDescriptionHTML, present := fields["description_html"]
		if !present {
			taskDescriptionHTML = description.html
		}
		if err := repository.admitPageTransaction(ctx, taskDescriptionHTML, nil, page.String()); err != nil {
			return 0, nil, err
		}
	}
	return repository.pageLifecycleDetail(ctx, user, slug, project, page, timezone, http.StatusCreated, "create")
}

func (repository *Repository) validatePageCreateFields(ctx context.Context, workspace, project uuid.UUID, input djangoinput.Input, handlerParent *uuid.UUID) (pageCreateFields, error) {
	fields := input.Fields
	name := ""
	validationErrors := map[string]any{}
	if value, present := input.FieldValue("name", false, true, false); present {
		var failures []string
		name, failures = djangoinput.TextField(value, true, 0, input.Surrogates["name"])
		if len(failures) > 0 {
			validationErrors["name"] = failures
		}
	}
	color := ""
	if value, present := input.FieldValue("color", false, true, false); present {
		var failures []string
		color, failures = djangoinput.CharField(value, true, input.Surrogates["color"])
		if len(failures) > 0 {
			validationErrors["color"] = failures
		}
	}
	access := 0
	if value, present := input.FieldValue("access", false, false, false); present {
		accessValue, valid := lifecycleChoice(value)
		if !valid {
			message := lifecycleChoiceMessage(input, value)
			if value == nil {
				message = "This field may not be null."
			}
			validationErrors["access"] = []string{message}
		} else {
			access = accessValue
		}
	}
	labelValue, labelsPresent := input.ListValue("labels", false)
	labelList := lifecycleParseLabelList(labelValue, labelsPresent, input.JSONFieldSource("labels"))
	labelList, err := repository.validateLifecycleLabelReferences(ctx, workspace, labelList)
	if err != nil {
		return pageCreateFields{}, err
	}
	if labelList.errors != nil {
		validationErrors["labels"] = labelList.errors
	}
	var parent any
	if value, present := fields["parent"]; present {
		parentValue, message, parentErr := repository.lifecycleSerializerParent(ctx, workspace, project, value, input.JSONFieldSource("parent"), handlerParent)
		if parentErr != nil {
			return pageCreateFields{}, parentErr
		}
		if message != "" {
			validationErrors["parent"] = []string{message}
		} else {
			parent = parentValue
		}
	}
	viewPropsRaw, viewPropsErr := lifecycleJSONField(input, "view_props", map[string]any{"full_width": false})
	if viewPropsErr != nil {
		validationErrors["view_props"] = []string{lifecycleJSONFieldMessage(viewPropsErr)}
	}
	logoPropsRaw, logoPropsErr := lifecycleJSONField(input, "logo_props", map[string]any{})
	if logoPropsErr != nil {
		validationErrors["logo_props"] = []string{lifecycleJSONFieldMessage(logoPropsErr)}
	}
	isLocked := false
	if lockedValue, present := input.BooleanFieldValue("is_locked", false); present {
		locked, valid := lifecycleBoolean(lockedValue)
		if !valid {
			validationErrors["is_locked"] = []string{"Must be a valid boolean."}
		} else if locked {
			isLocked = true
		}
	}
	var archivedAt any
	if value, present := input.FieldValue("archived_at", false, false, true); present && value != nil {
		date, ok := value.(string)
		if !ok {
			validationErrors["archived_at"] = []string{"Date has wrong format. Use one of these formats instead: YYYY-MM-DD."}
		} else if _, err := time.Parse(time.DateOnly, date); err != nil {
			validationErrors["archived_at"] = []string{"Date has wrong format. Use one of these formats instead: YYYY-MM-DD."}
		} else {
			archivedAt = date
		}
	}
	if len(validationErrors) > 0 {
		return pageCreateFields{}, &djangoapi.HTTPError{Status: 400, Body: validationErrors}
	}
	return pageCreateFields{
		name: name, color: color, access: access, labels: labelList,
		parent: parent, viewProps: viewPropsRaw, logoProps: logoPropsRaw,
		locked: isLocked, archivedAt: archivedAt,
	}, nil
}

func parsePageCreateContent(fields map[string]any) (pageCreateContent, error) {
	descriptionHTMLValue, present := fields["description_html"]
	if !present {
		descriptionHTMLValue = "<p></p>"
	}
	// Create passes description_html through serializer context directly rather
	// than through a declared DRF field. Keep its raw value for storage and task
	// publication; in particular, form strings retain surrounding whitespace.
	descriptionHTML := descriptionHTMLValue
	if descriptionHTML != nil {
		if _, ok := descriptionHTML.(string); !ok {
			// The source model's save hook calls strip_tags on this raw context
			// value before persistence. Non-string values fail there and reach
			// the view's generic exception boundary without durable effects.
			return pageCreateContent{}, &LifecycleError{Status: http.StatusInternalServerError, Body: map[string]any{"error": "Something went wrong please try again later"}}
		}
	}
	descriptionJSON, present := fields["description_json"]
	if !present {
		descriptionJSON = map[string]any{}
	}
	var jsonBytes any
	if descriptionJSON != nil {
		encoded, err := json.Marshal(descriptionJSON)
		if err != nil {
			return pageCreateContent{}, err
		}
		jsonBytes = encoded
	}
	binary := fields["description_binary"]
	if binary != nil {
		if _, ok := binary.([]byte); !ok {
			// Create treats this value as raw model context, not the dedicated
			// endpoint's base64 serializer field. The source database adapter
			// rejects a request string here before any transaction commits.
			return pageCreateContent{}, &LifecycleError{Status: http.StatusInternalServerError, Body: map[string]any{"error": "Something went wrong please try again later"}}
		}
	}
	var descriptionStripped any
	if html, ok := descriptionHTML.(string); ok {
		if html != "" {
			descriptionStripped = content.StripTags(html)
		}
	}
	return pageCreateContent{html: descriptionHTML, json: jsonBytes, binary: binary, stripped: descriptionStripped}, nil
}
