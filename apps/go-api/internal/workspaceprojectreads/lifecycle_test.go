package workspaceprojectreads

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type projectLifecycleTxFake struct {
	pgx.Tx
	queries               []string
	execArgs              [][]any
	failStates            bool
	committed, rolledBack bool
}

type projectCoverAssetsFake struct {
	allowed   bool
	err       error
	requested []uuid.UUID
}

func (f *projectCoverAssetsFake) Validate(_ context.Context, workspace, project, id uuid.UUID) (bool, error) {
	f.requested = []uuid.UUID{workspace, project, id}
	return f.allowed, f.err
}

func TestSessionProjectFieldsRejectActorAndArchiveAssignmentsIncludingNull(t *testing.T) {
	repo, _ := NewLifecycleRepository(&projectLifecycleDBFake{}, &projectLifecycleEffectsFake{}, "", validProjectCover)
	for _, value := range []any{nil, "00000000-0000-4000-8000-000000000001"} {
		_, err := repo.validateProjectFields(context.Background(), "workspace", "project", djangoinput.Input{Fields: map[string]any{"created_by": value, "updated_by": value, "archived_at": value}, Surrogates: nil}, true, false, false)
		var response *djangoapi.HTTPError
		if !errors.As(err, &response) || response.Status != 400 || len(response.Body) != 3 {
			t.Fatalf("server-owned input %#v: %v", value, err)
		}
		for _, key := range []string{"created_by", "updated_by", "archived_at"} {
			if !reflect.DeepEqual(response.Body[key], []string{"This field is read-only."}) {
				t.Errorf("%s=%v", key, response.Body[key])
			}
		}
	}
}

func TestSessionProjectCoverPreservesNullAndValidatesExistingAssetScope(t *testing.T) {
	workspace, project, id := uuid.New(), uuid.New(), uuid.New()
	db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{
		projectLifecycleBoolRow(true),
		projectLifecycleBoolRow(true),
		projectLifecycleBoolRow(true),
	}}}
	repo, _ := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
	reader := &projectCoverAssetsFake{allowed: true}
	repo.coverAssets = reader.Validate
	values, err := repo.validateProjectFields(context.Background(), workspace.String(), project.String(), djangoinput.Input{Fields: map[string]any{"cover_image_asset": nil}, Surrogates: nil}, true, false, false)
	if err != nil || values["cover_image_asset_id"] != nil || len(reader.requested) != 0 {
		t.Fatalf("null cover=%v,%v lookup=%v", values, err, reader.requested)
	}
	values, err = repo.validateProjectFields(context.Background(), workspace.String(), project.String(), djangoinput.Input{Fields: map[string]any{"cover_image_asset": id.String()}, Surrogates: nil}, true, false, false)
	if err != nil || values["cover_image_asset_id"] != id.String() || !slices.Equal(reader.requested, []uuid.UUID{workspace, project, id}) {
		t.Fatalf("scoped cover=%v,%v lookup=%v", values, err, reader.requested)
	}
	reader.allowed = false
	_, err = repo.validateProjectFields(context.Background(), workspace.String(), project.String(), djangoinput.Input{Fields: map[string]any{"cover_image_asset": id.String()}, Surrogates: nil}, true, false, false)
	var rejected *djangoapi.HTTPError
	if !errors.As(err, &rejected) || rejected.Status != 400 || rejected.Body["cover_image_asset"] == nil {
		t.Fatalf("invalid cover: %v", err)
	}

	reader.err = nil
	_, err = repo.validateProjectFields(context.Background(), workspace.String(), project.String(), djangoinput.Input{Fields: map[string]any{"cover_image_asset": id.String()}, Surrogates: nil}, true, false, false)
	var response *djangoapi.HTTPError
	if !errors.As(err, &response) || response.Status != 400 {
		t.Fatalf("foreign/missing cover=%v", err)
	}
}

func TestProjectTimezoneChoiceSetMatchesPinnedSourceAndRejectsSessionAliases(t *testing.T) {
	raw, err := os.ReadFile("../../projectlifecycle/source/source.openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var source struct {
		Components struct {
			Schemas struct {
				Timezone struct {
					Enum []string `json:"enum"`
				} `json:"TimezoneEnum"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err = json.Unmarshal(raw, &source); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(projectTimezones, source.Components.Schemas.Timezone.Enum) || len(projectTimezones) != 433 {
		t.Fatal("project timezone choices drifted from pinned source")
	}
	for _, zone := range []string{"UTC", "GMT", "US/Pacific", "Asia/Kolkata"} {
		if !validProjectTimezone(zone) {
			t.Errorf("rejected %s", zone)
		}
	}
	for _, zone := range []string{"Etc/UTC", "Asia/Calcutta", "CET", "Local", " UTC "} {
		if validProjectTimezone(zone) {
			t.Errorf("accepted %s", zone)
		}
	}
}

func (f *projectLifecycleTxFake) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	f.queries = append(f.queries, query)
	f.execArgs = append(f.execArgs, args)
	if f.failStates && strings.HasPrefix(query, "INSERT INTO states") {
		return pgconn.CommandTag{}, errors.New("state insert rejected")
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func TestProjectIdentifierUppercaseUsesFullUnicodeMappings(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"straße", "STRASSE"},
		{"ﬃ", "FFI"},
	} {
		if got := normalizeProjectIdentifier(test.input); got != test.want {
			t.Errorf("normalizeProjectIdentifier(%q)=%q, want %s", test.input, got, test.want)
		}
	}
}

func TestCreateProjectUppercasesUnicodeIdentifierWithFullMappings(t *testing.T) {
	tx := &projectLifecycleTxFake{}
	db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{
		projectLifecycleWorkspaceRow(),
		projectLifecycleBoolRow(false), // reservation
		workspaceProjectJSONRow(`{"id":"00000000-0000-4000-8000-000000000003","identifier":"STRASSE"}`),
	}}, tx: tx}
	effects := &projectLifecycleEffectsFake{}
	repo, _ := NewLifecycleRepository(db, effects, "", validProjectCover)
	input, err := djangoinput.Parse(strings.NewReader(`{"name":"Unicode Example","identifier":"straße"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.CreateProject(context.Background(), "00000000-0000-4000-8000-000000000001", "space", input, false); err != nil {
		t.Fatal(err)
	}
	if len(tx.execArgs) == 0 || len(tx.execArgs[0]) != 1 {
		t.Fatalf("project insert args=%v", tx.execArgs)
	}
	var stored map[string]any
	if err = json.Unmarshal([]byte(tx.execArgs[0][0].(string)), &stored); err != nil {
		t.Fatal(err)
	}
	if stored["identifier"] != "STRASSE" {
		t.Fatalf("stored identifier=%v, want STRASSE", stored["identifier"])
	}
	if len(tx.execArgs) < 2 || tx.execArgs[1][2] != "STRASSE" {
		t.Fatalf("ProjectIdentifier reservation args=%v, want STRASSE", tx.execArgs)
	}
	if len(db.args) < 2 || db.args[1][1] != "STRASSE" {
		t.Fatalf("reservation lookup args=%v, want STRASSE", db.args)
	}
	assertProjectRequestedDataJSON(t, effects, `{"name":"Unicode Example","identifier":"straße"}`)
}

func TestProjectPATCHUppercasesUnicodeIdentifierAndRetainsReservation(t *testing.T) {
	const projectID = "00000000-0000-4000-8000-000000000003"
	db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{
		workspaceProjectJSONRow(`{"workspace":"00000000-0000-4000-8000-000000000002","archived_at":null,"intake_view":false}`),
		workspaceProjectJSONRow(`{"id":"` + projectID + `","identifier":"STRASSE"}`),
	}}}
	effects := &projectLifecycleEffectsFake{}
	repo, _ := NewLifecycleRepository(db, effects, "", validProjectCover)
	input, err := djangoinput.Parse(strings.NewReader(`{"identifier":"straße"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.UpdateProject(context.Background(), "00000000-0000-4000-8000-000000000001", "space", projectID, input, false, false); err != nil {
		t.Fatal(err)
	}
	if len(db.execArgs) != 1 || len(db.execArgs[0]) != 2 {
		t.Fatalf("project update args=%v", db.execArgs)
	}
	var stored map[string]any
	if err = json.Unmarshal([]byte(db.execArgs[0][0].(string)), &stored); err != nil {
		t.Fatal(err)
	}
	if stored["identifier"] != "STRASSE" {
		t.Fatalf("PATCH stored identifier=%v, want STRASSE", stored["identifier"])
	}
	if strings.Contains(strings.Join(db.execs, "\n"), "project_identifiers") {
		t.Fatalf("PATCH changed identifier reservations: %v", db.execs)
	}
	assertProjectRequestedDataJSON(t, effects, `{"identifier":"straße"}`)
}

func TestProjectPUTUppercasesUnicodeIdentifierAndRetainsReservation(t *testing.T) {
	const projectID = "00000000-0000-4000-8000-000000000003"
	db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{
		workspaceProjectJSONRow(`{"workspace":"00000000-0000-4000-8000-000000000002","archived_at":null,"intake_view":false}`),
		projectLifecycleBoolRow(false), // name uniqueness
		projectLifecycleBoolRow(false), // identifier uniqueness
		workspaceProjectJSONRow(`{"id":"` + projectID + `","identifier":"STRASSE"}`),
	}}}
	effects := &projectLifecycleEffectsFake{}
	repo, _ := NewLifecycleRepository(db, effects, "", validProjectCover)
	input, err := djangoinput.Parse(strings.NewReader(`{"name":"Unicode project","identifier":"straße"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.UpdateProject(context.Background(), "00000000-0000-4000-8000-000000000001", "space", projectID, input, true, true); err != nil {
		t.Fatal(err)
	}
	if len(db.execArgs) != 1 || len(db.execArgs[0]) != 2 {
		t.Fatalf("project update args=%v", db.execArgs)
	}
	var stored map[string]any
	if err = json.Unmarshal([]byte(db.execArgs[0][0].(string)), &stored); err != nil {
		t.Fatal(err)
	}
	if stored["identifier"] != "STRASSE" {
		t.Fatalf("PUT stored identifier=%v, want STRASSE", stored["identifier"])
	}
	if strings.Contains(strings.Join(db.execs, "\n"), "project_identifiers") {
		t.Fatalf("PUT changed identifier reservations: %v", db.execs)
	}
	assertProjectRequestedDataJSON(t, effects, `{"name":"Unicode project","identifier":"straße"}`)
}

func TestProjectJSONFieldsNormalizeDecimalNumbersLikePython(t *testing.T) {
	repo, _ := NewLifecycleRepository(&projectLifecycleDBFake{}, &projectLifecycleEffectsFake{}, "", validProjectCover)
	input, err := djangoinput.Parse(strings.NewReader(`{"icon_prop":{"fraction":9007199254740993.0,"integer":9007199254740993,"text":"9007199254740993.0"}}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	values, err := repo.validateProjectFields(context.Background(), "workspace", "project", input, true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"fraction": json.Number("9007199254740992.0"),
		"integer":  json.Number("9007199254740993"),
		"text":     "9007199254740993.0",
	}
	if !reflect.DeepEqual(values["icon_prop"], want) {
		t.Fatalf("icon_prop=%#v, want %#v", values["icon_prop"], want)
	}

	formInput, err := djangoinput.Parse(strings.NewReader(`icon_prop=%7B%22n%22%3A9007199254740993.0%7D`), "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}
	formValues, err := repo.validateProjectFields(context.Background(), "workspace", "project", formInput, true, false, false)
	if err != nil || !reflect.DeepEqual(formValues["icon_prop"], map[string]any{"n": json.Number("9007199254740992.0")}) {
		t.Fatalf("form icon_prop=%#v, %v", formValues["icon_prop"], err)
	}
	requested, err := lifecycleRequestedData(formInput)
	if err != nil || !reflect.DeepEqual(requested, map[string]any{"icon_prop": `{"n":9007199254740993.0}`}) {
		t.Fatalf("form requested_data=%#v, %v", requested, err)
	}

	overflowInput, err := djangoinput.Parse(strings.NewReader(`{"icon_prop":{"n":1e309}}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.validateProjectFields(context.Background(), "workspace", "project", overflowInput, true, false, false)
	var response *djangoapi.HTTPError
	if !errors.As(err, &response) || response.Status != 400 || !reflect.DeepEqual(response.Body["icon_prop"], []string{"Value must be valid JSON."}) {
		t.Fatalf("overflow JSONField error=%v, want source JSONField validation error", err)
	}
}

func TestProjectJSONFieldUnicodeValidationUsesOriginalJSONAndFormSource(t *testing.T) {
	tests := []struct {
		name, body, contentType string
		valid                   bool
		wantValue               any
	}{
		{name: "unpaired high surrogate value", body: `{"icon_prop":{"value":"\ud800"}}`, contentType: "application/json"},
		{name: "unpaired low surrogate value", body: `{"icon_prop":{"value":"\udc00"}}`, contentType: "application/json"},
		{name: "unpaired surrogate object key", body: `{"icon_prop":{"\ud800":"value"}}`, contentType: "application/json"},
		{name: "invalid surrogate in overwritten duplicate", body: `{"icon_prop":{"key":"\ud800","key":"last"}}`, contentType: "application/json"},
		{name: "form unpaired surrogate", body: `icon_prop=%7B%22value%22%3A%22%5Cud800%22%7D`, contentType: "application/x-www-form-urlencoded"},
		{name: "valid surrogate pair", body: `{"icon_prop":{"value":"\ud83d\ude00"}}`, contentType: "application/json", valid: true},
		{name: "literal replacement character", body: `{"icon_prop":{"value":"�"}}`, contentType: "application/json", valid: true},
		{name: "valid duplicate final value", body: `{"icon_prop":{"key":"first","key":"last"}}`, contentType: "application/json", valid: true, wantValue: map[string]any{"key": "last"}},
		{name: "form valid surrogate pair", body: `icon_prop=%7B%22value%22%3A%22%5Cud83d%5Cude00%22%7D`, contentType: "application/x-www-form-urlencoded", valid: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input, err := djangoinput.Parse(strings.NewReader(test.body), test.contentType)
			if err != nil {
				t.Fatal(err)
			}
			repo, _ := NewLifecycleRepository(&projectLifecycleDBFake{}, &projectLifecycleEffectsFake{}, "", validProjectCover)
			values, err := repo.validateProjectFields(context.Background(), "workspace", "project", input, false, false, false)
			if test.valid {
				if err != nil {
					t.Fatalf("valid JSONField rejected: %v", err)
				}
				if test.wantValue != nil && !reflect.DeepEqual(values["icon_prop"], test.wantValue) {
					t.Fatalf("icon_prop=%#v, want %#v", values["icon_prop"], test.wantValue)
				}
				return
			}
			var response *djangoapi.HTTPError
			if !errors.As(err, &response) || response.Status != 400 || !reflect.DeepEqual(response.Body["icon_prop"], []string{"Value must be valid JSON."}) {
				t.Fatalf("invalid JSONField error=%v, want field-shaped 400", err)
			}
		})
	}
}

func TestSessionProjectJSONFieldUnicodeValidationCoversAllPersistedFields(t *testing.T) {
	for _, field := range []string{"icon_prop", "logo_props", "description_text", "description_html"} {
		t.Run(field, func(t *testing.T) {
			body := `{"` + field + `":{"value":"\ud800"}}`
			input, err := djangoinput.Parse(strings.NewReader(body), "application/json")
			if err != nil {
				t.Fatal(err)
			}
			repo, _ := NewLifecycleRepository(&projectLifecycleDBFake{}, &projectLifecycleEffectsFake{}, "", validProjectCover)
			_, err = repo.validateProjectFields(context.Background(), "workspace", "project", input, true, false, false)
			var response *djangoapi.HTTPError
			if !errors.As(err, &response) || response.Status != 400 || !reflect.DeepEqual(response.Body[field], []string{"Value must be valid JSON."}) {
				t.Fatalf("session %s invalid JSONField error=%v, want field-shaped 400", field, err)
			}
		})
	}
}

func TestAPIProjectValidationIgnoresNonIconJSONFieldUnicode(t *testing.T) {
	input, err := djangoinput.Parse(strings.NewReader(`{"description_html":{"value":"\ud800"},"description_text":{"value":"\ud800"},"logo_props":{"value":"\ud800"}}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	repo, _ := NewLifecycleRepository(&projectLifecycleDBFake{}, &projectLifecycleEffectsFake{}, "", validProjectCover)
	values, err := repo.validateProjectFields(context.Background(), "workspace", "project", input, false, false, false)
	if err != nil {
		t.Fatalf("API ignored non-icon JSONFields were rejected: %v", err)
	}
	for _, field := range []string{"description_html", "description_text", "logo_props"} {
		if _, present := values[field]; present {
			t.Errorf("API persisted ignored field %q: %#v", field, values[field])
		}
	}
}

func TestProjectCreateAndPATCHRejectUnpairedJSONFieldBeforeWritesOrActivity(t *testing.T) {
	const projectID = "00000000-0000-4000-8000-000000000003"
	const body = `{"name":"Unicode Example","identifier":"UNIC","icon_prop":{"value":"\ud800"}}`

	for _, operation := range []string{"create", "PATCH"} {
		t.Run(operation, func(t *testing.T) {
			var db *projectLifecycleDBFake
			var tx *projectLifecycleTxFake
			if operation == "create" {
				tx = &projectLifecycleTxFake{}
				db = &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{projectLifecycleWorkspaceRow()}}, tx: tx}
			} else {
				db = &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{
					workspaceProjectJSONRow(`{"id":"` + projectID + `","workspace":"00000000-0000-4000-8000-000000000002","archived_at":null,"intake_view":false}`),
				}}}
			}
			effects := &projectLifecycleEffectsFake{}
			repo, _ := NewLifecycleRepository(db, effects, "", validProjectCover)
			input, err := djangoinput.Parse(strings.NewReader(body), "application/json")
			if err != nil {
				t.Fatal(err)
			}
			if operation == "create" {
				_, err = repo.CreateProject(context.Background(), "00000000-0000-4000-8000-000000000001", "space", input, false)
			} else {
				_, err = repo.UpdateProject(context.Background(), "00000000-0000-4000-8000-000000000001", "space", projectID, input, false, false)
			}
			var response *djangoapi.HTTPError
			if !errors.As(err, &response) || response.Status != 400 || response.Body["icon_prop"] == nil {
				t.Fatalf("%s invalid JSONField error=%v, want field-shaped 400", operation, err)
			}
			if len(db.execArgs) != 0 || len(effects.payloads) != 0 || tx != nil && (tx.committed || len(tx.execArgs) != 0) {
				t.Fatalf("%s invalid JSONField wrote data or published activity: DB=%v tx=%+v effects=%v", operation, db.execArgs, tx, effects.payloads)
			}
		})
	}
}

func TestLifecycleRequestedDataUsesPythonNumbersWithoutMutatingInput(t *testing.T) {
	const raw = `{"icon_prop":{"fraction":9007199254740993.0,"integer":9007199254740993,"text":"9007199254740993.0"},"unused":1e309}`
	input, err := djangoinput.Parse(strings.NewReader(raw), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	requested, err := lifecycleRequestedData(input)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(requested)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"icon_prop":{"fraction":9007199254740992.0,"integer":9007199254740993,"text":"9007199254740993.0"},"unused":1e309}`
	if string(encoded) != want {
		t.Fatalf("requested_data=%s, want %s", encoded, want)
	}
	original := input.Fields["icon_prop"].(map[string]any)
	if original["fraction"] != json.Number("9007199254740993.0") {
		t.Fatalf("request input was mutated: %#v", original)
	}
	if string(input.Raw) != raw || input.Fields["unused"] != json.Number("1e309") {
		t.Fatalf("raw request changed: raw=%s fields=%#v", input.Raw, input.Fields)
	}
}

func TestProjectCreateAndPATCHActivityPreserveDecodedJSONUnicodeAndNumbers(t *testing.T) {
	const projectID = "00000000-0000-4000-8000-000000000003"
	const raw = `{"name":"Unicode Example","identifier":"UNIC","unused":{"\ud800":"surrogate","\ufffd":"replacement","value":"\ud800","astral":{"\ud83d\ude00":"first","😀":"last"},"fraction":9007199254740993.0,"overflow":1e309}}`
	const wantRequestedData = `{"name":"Unicode Example","identifier":"UNIC","unused":{"\ud800":"surrogate","\ufffd":"replacement","value":"\ud800","astral":{"\ud83d\ude00":"first","😀":"last"},"fraction":9007199254740992.0,"overflow":1e309}}`

	for _, operation := range []string{"create", "PATCH"} {
		t.Run(operation, func(t *testing.T) {
			var db *projectLifecycleDBFake
			if operation == "create" {
				db = &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{
					projectLifecycleWorkspaceRow(),
					projectLifecycleBoolRow(false),
					workspaceProjectJSONRow(`{"id":"` + projectID + `","identifier":"UNIC"}`),
				}}, tx: &projectLifecycleTxFake{}}
			} else {
				db = &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{
					workspaceProjectJSONRow(`{"id":"` + projectID + `","workspace":"00000000-0000-4000-8000-000000000002","archived_at":null,"intake_view":false}`),
					workspaceProjectJSONRow(`{"id":"` + projectID + `","identifier":"UNIC"}`),
				}}}
			}
			effects := &projectLifecycleEffectsFake{}
			repo, err := NewLifecycleRepository(db, effects, "", validProjectCover)
			if err != nil {
				t.Fatal(err)
			}
			input, err := djangoinput.Parse(strings.NewReader(raw), "application/json")
			if err != nil {
				t.Fatal(err)
			}
			if operation == "create" {
				_, err = repo.CreateProject(context.Background(), "00000000-0000-4000-8000-000000000001", "space", input, false)
			} else {
				_, err = repo.UpdateProject(context.Background(), "00000000-0000-4000-8000-000000000001", "space", projectID, input, false, false)
			}
			if err != nil {
				t.Fatalf("successful %s: %v", operation, err)
			}
			if len(effects.payloads) != 1 {
				t.Fatalf("activity payload count=%d, want 1", len(effects.payloads))
			}
			taskBody, err := json.Marshal([]any{[]any{}, effects.payloads[0], map[string]any{
				"callbacks": nil, "errbacks": nil, "chain": nil, "chord": nil,
			}})
			if err != nil {
				t.Fatal(err)
			}
			wantFragment := `"requested_data":` + wantRequestedData
			if !strings.Contains(string(taskBody), wantFragment) {
				t.Fatalf("%s activity task body lost source JSON distinctions:\n got: %s\nwant fragment: %s", operation, taskBody, wantFragment)
			}
			var envelope []json.RawMessage
			if err = json.Unmarshal(taskBody, &envelope); err != nil {
				t.Fatal(err)
			}
			var kwargs map[string]json.RawMessage
			if err = json.Unmarshal(envelope[1], &kwargs); err != nil {
				t.Fatal(err)
			}
			var requested map[string]json.RawMessage
			if err = json.Unmarshal(kwargs["requested_data"], &requested); err != nil {
				t.Fatal(err)
			}
			var unused map[string]json.RawMessage
			if err = json.Unmarshal(requested["unused"], &unused); err != nil {
				t.Fatal(err)
			}
			var astral map[string]json.RawMessage
			if err = json.Unmarshal(unused["astral"], &astral); err != nil {
				t.Fatal(err)
			}
			if string(astral["😀"]) != `"last"` {
				t.Fatalf("%s decoded duplicate astral key=%s, want last value", operation, astral["😀"])
			}
			if string(input.Raw) != raw {
				t.Fatalf("%s raw audit body changed: %s", operation, input.Raw)
			}
		})
	}
}

func TestProjectCompositeValidationErrorsUsePythonSourceRepresentation(t *testing.T) {
	tests := []struct {
		name, field, body, want string
		app                     bool
	}{
		{
			name:  "project lead preserves order, Unicode printability, and surrogates",
			field: "project_lead",
			body:  `{"project_lead":{"z":1,"a":["\u00a0","\ufffd","\ud833\udc00","\ud800"]}}`,
			want:  `“{'z': 1, 'a': ['\xa0', '�', '\U0001cc00', '\ud800']}” is not a valid UUID.`,
		},
		{
			name:  "default assignee preserves nested order",
			field: "default_assignee",
			body:  `{"default_assignee":["\u00a0",{"b":2,"a":true}]}`,
			want:  `“['\xa0', {'b': 2, 'a': True}]” is not a valid UUID.`,
		},
		{
			name:  "default state preserves nested order",
			field: "default_state",
			body:  `{"default_state":{"state":[1,{"z":"\u00a0"}]}}`,
			want:  `“{'state': [1, {'z': '\xa0'}]}” is not a valid UUID.`,
		},
		{
			name:  "estimate list",
			field: "estimate",
			body:  `{"estimate":["point",{"z":0}]}`,
			want:  `“['point', {'z': 0}]” is not a valid UUID.`,
		},
		{
			name:  "timezone choice",
			field: "timezone",
			body:  `{"timezone":["UTC",{"tz":"\u00a0"}]}`,
			want:  `"['UTC', {'tz': '\xa0'}]" is not a valid choice.`,
		},
		{
			name:  "network choice",
			field: "network",
			body:  `{"network":{"z":1,"a":[false]}}`,
			want:  `"{'z': 1, 'a': [False]}" is not a valid choice.`,
			app:   true,
		},
		{
			name:  "cover asset UUID",
			field: "cover_image_asset",
			body:  `{"cover_image_asset":{"z":"asset","a":[true,1]}}`,
			want:  `“{'z': 'asset', 'a': [True, 1]}” is not a valid UUID.`,
			app:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input, err := djangoinput.Parse(strings.NewReader(test.body), "application/json")
			if err != nil {
				t.Fatal(err)
			}
			repo, _ := NewLifecycleRepository(&projectLifecycleDBFake{}, &projectLifecycleEffectsFake{}, "", validProjectCover)
			_, err = repo.validateProjectFields(context.Background(), "workspace", "project", input, test.app, false, false)
			var response *djangoapi.HTTPError
			if !errors.As(err, &response) || response.Status != 400 || !reflect.DeepEqual(response.Body[test.field], []string{test.want}) {
				t.Fatalf("validation response=%v, want %s=%q", err, test.field, test.want)
			}
		})
	}
}

func TestProjectForeignKeyValidationKeepsSourceTypeAndUUIDErrors(t *testing.T) {
	missing := "00000000-0000-4000-8000-0000000000AB"
	tests := []struct {
		name, field string
		value       any
		want        string
		rows        []pgx.Row
		queries     int
	}{
		{name: "boolean", field: "project_lead", value: true, want: "Incorrect type. Expected pk value, received bool."},
		{name: "malformed UUID", field: "project_lead", value: "not-a-uuid", want: "“not-a-uuid” is not a valid UUID."},
		{name: "missing UUID preserves spelling", field: "estimate", value: missing, want: `Invalid pk "` + missing + `" - object does not exist.`, rows: []pgx.Row{projectLifecycleBoolRow(false)}, queries: 1},
		{name: "malformed cover UUID", field: "cover_image_asset", value: "not-a-uuid", want: "“not-a-uuid” is not a valid UUID."},
		{name: "boolean cover asset", field: "cover_image_asset", value: true, want: "Incorrect type. Expected pk value, received bool."},
		{name: "missing cover asset preserves spelling", field: "cover_image_asset", value: missing, want: `Invalid pk "` + missing + `" - object does not exist.`, rows: []pgx.Row{projectLifecycleBoolRow(false)}, queries: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: test.rows}}
			repo, _ := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
			_, err := repo.validateProjectFields(context.Background(), uuid.NewString(), uuid.NewString(), djangoinput.Input{Fields: map[string]any{test.field: test.value}}, true, false, false)
			var response *djangoapi.HTTPError
			if !errors.As(err, &response) || response.Status != 400 || !reflect.DeepEqual(response.Body[test.field], []string{test.want}) {
				t.Fatalf("validation error=%v, want %s", err, test.want)
			}
			if len(db.queries) != test.queries {
				t.Fatalf("queries=%d, want %d", len(db.queries), test.queries)
			}
		})
	}
}

func TestProjectBooleanAcceptsNumericNegativeZeroButRejectsStringControl(t *testing.T) {
	for _, raw := range []json.Number{"-0.0", "-0e2", "-1e-9999"} {
		if got, ok := projectBoolean(raw); !ok || got {
			t.Errorf("projectBoolean(%q)=(%v,%v), want (false,true)", raw, got, ok)
		}
	}
	if got, ok := projectBoolean("-0.0"); ok {
		t.Errorf("projectBoolean(string negative zero)=(%v,%v), want rejected", got, ok)
	}

	repo, _ := NewLifecycleRepository(&projectLifecycleDBFake{}, &projectLifecycleEffectsFake{}, "", validProjectCover)
	for _, body := range []string{`{"page_view":-0.0}`, `{"page_view":-0e2}`, `{"page_view":-1e-9999}`} {
		input, err := djangoinput.Parse(strings.NewReader(body), "application/json")
		if err != nil {
			t.Fatal(err)
		}
		values, err := repo.validateProjectFields(context.Background(), "workspace", "project", input, true, false, false)
		if err != nil || values["page_view"] != false {
			t.Errorf("validate %s=(%v,%v), want false", body, values["page_view"], err)
		}
	}
	input, err := djangoinput.Parse(strings.NewReader(`{"page_view":"-0.0"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.validateProjectFields(context.Background(), "workspace", "project", input, true, false, false)
	var response *djangoapi.HTTPError
	if !errors.As(err, &response) || response.Body["page_view"] == nil {
		t.Fatalf("string negative zero error=%v, want boolean field error", err)
	}
}
func (f *projectLifecycleTxFake) Commit(context.Context) error   { f.committed = true; return nil }
func (f *projectLifecycleTxFake) Rollback(context.Context) error { f.rolledBack = true; return nil }

type projectLifecycleDBFake struct {
	workspaceProjectDatabase
	tx       *projectLifecycleTxFake
	execs    []string
	execArgs [][]any
}

func (f *projectLifecycleDBFake) Begin(context.Context) (pgx.Tx, error) { return f.tx, nil }
func (f *projectLifecycleDBFake) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	f.execs = append(f.execs, q)
	f.execArgs = append(f.execArgs, args)
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

type projectLifecycleEffectsFake struct {
	tasks      []string
	softDelete []string
	payloads   []map[string]any
}

func (f *projectLifecycleEffectsFake) Task(_ context.Context, name string, _ []any, kwargs map[string]any) error {
	f.tasks = append(f.tasks, name)
	f.payloads = append(f.payloads, kwargs)
	return nil
}

func assertProjectRequestedDataJSON(t *testing.T, effects *projectLifecycleEffectsFake, want string) {
	t.Helper()
	if len(effects.payloads) != 1 {
		t.Fatalf("activity payload count=%d, want 1", len(effects.payloads))
	}
	encoded, err := json.Marshal(effects.payloads[0]["requested_data"])
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != want {
		t.Fatalf("activity request_data=%s, want %s", encoded, want)
	}
}

func TestProjectHTMLMatchesSourceContainersAndEmptyJSONValues(t *testing.T) {
	repo, _ := NewLifecycleRepository(&projectLifecycleDBFake{}, &projectLifecycleEffectsFake{}, "", validProjectCover)
	for _, vector := range []struct {
		body string
		want any
	}{
		{`{"description_html":{"type":"doc","content":[{"type":"paragraph"}]}}`, `{'type': 'doc', 'content': [{'type': 'paragraph'}]}`},
		{`{"description_html":["<p>first</p>","<p>second</p>"]}`, `['<p>first</p>', '<p>second</p>']`},
		{`{"description_html":[]}`, []any{}},
		{`{"description_html":{}}`, map[string]any{}},
		{`{"description_html":0}`, json.Number("0")},
		{`{"description_html":false}`, false},
	} {
		input, err := djangoinput.Parse(strings.NewReader(vector.body), "application/json")
		if err != nil {
			t.Fatal(err)
		}
		values, err := repo.validateProjectFields(context.Background(), "workspace", "project", input, true, false, false)
		if err != nil || !reflect.DeepEqual(values["description_html"], vector.want) {
			t.Errorf("HTML %s=%#v,%v want %#v", vector.body, values["description_html"], err, vector.want)
		}
	}
}

func TestProjectFullAndPartialFormsUseLastValueJSONAndCheckboxSemantics(t *testing.T) {
	for _, replace := range []bool{false, true} {
		db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{projectLifecycleBoolRow(false), projectLifecycleBoolRow(false)}}}
		repo, _ := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
		input, err := djangoinput.Parse(strings.NewReader(`name=first&name=last&identifier=FORM&description_html=%7B%22type%22%3A%22doc%22%7D`), "application/x-www-form-urlencoded")
		if err != nil {
			t.Fatal(err)
		}
		values, err := repo.validateProjectFields(context.Background(), "workspace", "project", input, true, replace, false)
		if err != nil || values["name"] != "last" || values["description_html"] != `{'type': 'doc'}` {
			t.Fatalf("form values=%v,%v", values, err)
		}
		value, present := values["page_view"]
		if replace && (!present || value != false) || !replace && present {
			t.Errorf("replace=%v checkbox=%v,%v", replace, value, present)
		}
		if input.Fields["description_html"] != `{"type":"doc"}` {
			t.Fatal("form task/request data was normalized")
		}
	}
}

func TestProjectIntakeEffectUsesValidatedBooleanAndRetainsOriginalTaskInput(t *testing.T) {
	db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{
		workspaceProjectJSONRow(`{"workspace":"00000000-0000-4000-8000-000000000002","archived_at":null,"intake_view":false}`),
		workspaceProjectJSONRow(`{"id":"00000000-0000-4000-8000-000000000003","sort_order":65535}`),
	}}}
	effects := &projectLifecycleEffectsFake{}
	repo, _ := NewLifecycleRepository(db, effects, "", validProjectCover)
	input, err := djangoinput.Parse(strings.NewReader(`inbox_view=false`), "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.UpdateProject(context.Background(), "00000000-0000-4000-8000-000000000001", "space", "00000000-0000-4000-8000-000000000003", input, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(db.execs) != 1 || strings.Contains(db.execs[0], "INSERT INTO intakes") {
		t.Fatalf("false input created Intake: %v", db.execs)
	}
	var stored map[string]any
	if err = json.Unmarshal([]byte(db.execArgs[0][0].(string)), &stored); err != nil || stored["intake_view"] != false {
		t.Fatalf("stored=%v,%v", stored, err)
	}
	if len(effects.payloads) != 1 || !reflect.DeepEqual(effects.payloads[0]["requested_data"], map[string]any{"inbox_view": "false"}) {
		t.Fatalf("task input=%v", effects.payloads)
	}
}
func (f *projectLifecycleEffectsFake) SoftDelete(_ context.Context, app, model, id string) error {
	f.softDelete = append(f.softDelete, app+":"+model+":"+id)
	return nil
}
func projectLifecycleBoolRow(value bool) pgx.Row {
	return workspaceProjectRow(func(dest ...any) error { *dest[0].(*bool) = value; return nil })
}
func projectLifecycleWorkspaceRow() pgx.Row {
	return workspaceProjectRow(func(dest ...any) error {
		*dest[0].(*string) = "00000000-0000-4000-8000-000000000002"
		*dest[1].(*string) = "UTC"
		return nil
	})
}

func TestCreateProjectRollsBackMembershipAndStatesBeforePublishing(t *testing.T) {
	db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{projectLifecycleWorkspaceRow(), projectLifecycleBoolRow(false)}}, tx: &projectLifecycleTxFake{failStates: true}}
	effects := &projectLifecycleEffectsFake{}
	repo, _ := NewLifecycleRepository(db, effects, "http://localhost", validProjectCover)
	_, err := repo.CreateProject(context.Background(), "00000000-0000-4000-8000-000000000001", "space", djangoinput.Input{Fields: map[string]any{"name": "Created", "identifier": "NEW"}, Surrogates: nil}, false)
	if err == nil {
		t.Fatal("creation succeeded after rejected state insert")
	}
	if db.tx.committed || !db.tx.rolledBack {
		t.Fatalf("committed=%v rolledBack=%v", db.tx.committed, db.tx.rolledBack)
	}
	if len(effects.tasks) != 0 {
		t.Fatalf("published tasks before commit: %v", effects.tasks)
	}
	if strings.Contains(db.tx.queries[1], "created_by_id") {
		t.Fatalf("AuditModel reservation received BaseModel actor assignment: %s", db.tx.queries[1])
	}
	joined := strings.Join(db.tx.queries, "\n")
	if !strings.Contains(joined, "INSERT INTO project_identifiers") || !strings.Contains(joined, "INSERT INTO project_user_properties") || !strings.Contains(joined, "INSERT INTO project_members") {
		t.Fatalf("creation side effects omitted: %s", joined)
	}
}
func TestProjectWritesRequireActiveWorkspaceBeforeProjectAdmin(t *testing.T) {
	db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{projectLifecycleBoolRow(false)}}}
	repo, _ := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
	err := repo.AuthorizeProjectWrite(context.Background(), "actor", "space", "project", "patch", true)
	if err == nil {
		t.Fatal("stale project admin authorized")
	}
	if !strings.Contains(db.queries[0], "wm.is_active AND wm.deleted_at IS NULL") || !strings.Contains(db.queries[0], "wm.role=20 OR EXISTS") {
		t.Fatalf("missing membership or admin shortcut: %s", db.queries[0])
	}
}

func TestAPIProjectAuthorizationRejectsAbsentProjectScopeBeforeUUIDQuery(t *testing.T) {
	db := &projectLifecycleDBFake{}
	repo, _ := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
	// ProjectBasePermission filters project_id=None for unsupported unsafe
	// collection methods. No membership can match that absent project scope.
	if err := repo.AuthorizeProjectWrite(context.Background(), "actor", "space", "", "patch", false); !errors.Is(err, ErrForbidden) {
		t.Fatalf("absent project scope authorization=%v, want forbidden", err)
	}
	if len(db.queries) != 0 {
		t.Fatalf("absent project scope reached UUID membership lookup: %v", db.queries)
	}
}

func TestAPIProjectPATCHMissingProjectRetainsHandlerErrorBody(t *testing.T) {
	for _, checkOnly := range []bool{false, true} {
		db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{
			workspaceProjectRow(func(...any) error { return pgx.ErrNoRows }),
		}}}
		repo, _ := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
		var err error
		if checkOnly {
			err = repo.CheckProjectMutation(context.Background(), "space", "project", "patch", false)
		} else {
			_, err = repo.UpdateProject(context.Background(), "actor", "space", "project", djangoinput.Input{Fields: map[string]any{}}, false, false)
		}
		var failure *djangoapi.HTTPError
		if !errors.As(err, &failure) || failure.Status != 404 || !reflect.DeepEqual(failure.Body, map[string]any{"error": "Project does not exist"}) {
			t.Errorf("checkOnly=%v missing API PATCH=%v", checkOnly, err)
		}
	}
}
func TestProjectRestoreRetainsProjectMembershipWhileV1ArchiveUsesWorkspaceRole(t *testing.T) {
	db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{projectLifecycleBoolRow(true), projectLifecycleBoolRow(true)}}}
	repo, _ := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
	if err := repo.AuthorizeProjectWrite(context.Background(), "actor", "space", "project", "archive", false); err != nil {
		t.Fatal(err)
	}
	if err := repo.AuthorizeProjectWrite(context.Background(), "actor", "space", "project", "restore", false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(db.queries[0], "project_members") || !strings.Contains(db.queries[1], "project_members") {
		t.Fatalf("POST/DELETE permission branch collapsed: %v", db.queries)
	}
}
func TestProjectCurrentInstancePreservesIntegerAndFieldOrder(t *testing.T) {
	raw := []byte(`{"id":"p","archive_in":0,"network":2,"description":null}`)
	got, err := projectCurrentInstance(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"id": "p", "archive_in": 0, "network": 2, "description": null}` {
		t.Fatalf("current_instance=%s", got)
	}
}

func TestProjectWriteResponsePreservesSerializerFloatTypeWithoutChangingIntegerFields(t *testing.T) {
	for _, vector := range []struct{ input, want string }{
		{`{"sort_order":65535,"network":2}`, `{"network":2,"sort_order":65535.0}`},
		{`{"sort_order":1.25,"network":2}`, `{"network":2,"sort_order":1.25}`},
		{`{"sort_order":null,"network":2}`, `{"network":2,"sort_order":null}`},
	} {
		got, err := projectResponseFloatFields([]byte(vector.input))
		if err != nil || string(got) != vector.want {
			t.Errorf("response=%s,%v want %s", got, err, vector.want)
		}
	}
}

func TestDeleteProjectAppliesFullModelSaveAuditAndQueuesCascade(t *testing.T) {
	db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{projectLifecycleBoolRow(true)}}}
	effects := &projectLifecycleEffectsFake{}
	repo, _ := NewLifecycleRepository(db, effects, "http://localhost", validProjectCover)
	if err := repo.DeleteProject(context.Background(), "actor", "space", "project", true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(db.execs[0], "updated_at=clock_timestamp(),updated_by_id=$2") || len(db.execArgs[0]) != 2 || db.execArgs[0][1] != "actor" {
		t.Fatalf("tombstone missed model audit: %s args=%v", db.execs[0], db.execArgs[0])
	}
	if len(effects.softDelete) != 1 || effects.softDelete[0] != "db:project:project" {
		t.Fatalf("cascade=%v", effects.softDelete)
	}
	if len(effects.tasks) != 1 || effects.tasks[0] != "plane.bgtasks.webhook_task.webhook_activity" {
		t.Fatalf("tasks=%v", effects.tasks)
	}
}

func TestAPIProjectUpdateRejectsForeignStateWithFieldErrorBeforeWrite(t *testing.T) {
	db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{projectLifecycleBoolRow(true), projectLifecycleBoolRow(false)}}}
	repo, _ := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
	_, err := repo.validateProjectFields(context.Background(), "workspace", "project", djangoinput.Input{Fields: map[string]any{"default_state": "00000000-0000-4000-8000-000000000001"}, Surrogates: nil}, false, false, false)
	var response *djangoapi.HTTPError
	if !errors.As(err, &response) || response.Status != 400 || !reflect.DeepEqual(response.Body, map[string]any{"default_state": []string{"The provided state does not belong to this project."}}) {
		t.Fatalf("foreign state error=%v", err)
	}
	if len(db.execs) > 0 {
		t.Fatalf("foreign state wrote data: %v", db.execs)
	}
}
func TestAPIProjectUpdateAcceptsStateBelongingToProject(t *testing.T) {
	db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{projectLifecycleBoolRow(true), projectLifecycleBoolRow(true)}}}
	repo, _ := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
	values, err := repo.validateProjectFields(context.Background(), "workspace", "project", djangoinput.Input{Fields: map[string]any{"default_state": "00000000-0000-4000-8000-000000000001"}, Surrogates: nil}, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if values["default_state_id"] != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("own state=%v", values)
	}
}

func validProjectCover(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}

func TestNewSessionProjectRejectsExistingStateAndEstimateButAcceptsNull(t *testing.T) {
	for _, field := range []string{"default_state", "estimate"} {
		db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{projectLifecycleBoolRow(true), projectLifecycleBoolRow(false), projectLifecycleBoolRow(false), projectLifecycleBoolRow(false)}}}
		repo, _ := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
		fields := map[string]any{"name": "Created", "identifier": "NEW", field: uuid.NewString()}
		_, err := repo.validateProjectFields(context.Background(), uuid.NewString(), "", djangoinput.Input{Fields: fields, Surrogates: nil}, true, false, true)
		var response *djangoapi.HTTPError
		if !errors.As(err, &response) || response.Status != 400 || response.Body[field] == nil {
			t.Fatalf("new project %s=%v", field, err)
		}
		if len(db.execs) != 0 {
			t.Fatal("invalid relation persisted")
		}
		db.rows = []pgx.Row{projectLifecycleBoolRow(false), projectLifecycleBoolRow(false)}
		fields[field] = nil
		values, err := repo.validateProjectFields(context.Background(), uuid.NewString(), "", djangoinput.Input{Fields: fields, Surrogates: nil}, true, false, true)
		if err != nil || values[field+"_id"] != nil {
			t.Fatalf("nullable %s=%v,%v", field, values, err)
		}
	}
}

func TestProjectFieldValidationCollectsNullChoiceAndIntegerFailures(t *testing.T) {
	repo, _ := NewLifecycleRepository(&projectLifecycleDBFake{}, &projectLifecycleEffectsFake{}, "", validProjectCover)
	_, err := repo.validateProjectFields(context.Background(), "workspace", "project", djangoinput.Input{Fields: map[string]any{"page_view": nil, "archive_in": "1.5", "network": json.Number("2.0"), "timezone": "Etc/UTC"}, Surrogates: nil}, true, false, false)
	var response *djangoapi.HTTPError
	if !errors.As(err, &response) || response.Status != 400 || len(response.Body) != 4 {
		t.Fatalf("invalid bundle=%v", err)
	}
	values, err := repo.validateProjectFields(context.Background(), "workspace", "project", djangoinput.Input{Fields: map[string]any{"page_view": json.Number("1.00"), "archive_in": "1.000", "network": 2, "timezone": "US/Pacific"}, Surrogates: nil}, true, false, false)
	if err != nil || values["page_view"] != true || values["archive_in"] != int64(1) || values["network"] != int64(2) {
		t.Fatalf("valid coercion=%v,%v", values, err)
	}
}

func (f *projectLifecycleDBFake) BeginTx(ctx context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	return f.Begin(ctx)
}
