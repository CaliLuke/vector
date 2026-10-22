package modulereads

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestModuleActivitySnapshotUsesAppSerializerFieldOrder(t *testing.T) {
	values := map[string]any{
		"id": "module", "workspace_id": "workspace", "project_id": "project", "name": "Roadmap",
		"description": "", "description_text": nil, "description_html": nil, "start_date": nil,
		"target_date": nil, "status": "planned", "lead_id": nil, "member_ids": []any{},
		"view_props": map[string]any{"layout": "list"}, "sort_order": 65535.0,
		"external_source": nil, "external_id": nil, "logo_props": map[string]any{},
		"total_estimate_points": 0.0, "completed_estimate_points": 0.0, "is_favorite": false,
		"total_issues": json.Number("0"), "cancelled_issues": json.Number("0"),
		"completed_issues": json.Number("0"), "started_issues": json.Number("0"),
		"unstarted_issues": json.Number("0"), "backlog_issues": json.Number("0"),
		"created_at": "2026-10-20T01:00:00-07:00", "updated_at": "2026-10-20T01:00:00-07:00", "archived_at": nil,
	}
	// The input simulates an unordered JSON projection; serializer order must
	// come from the source ModuleSerializer declaration.
	original := []byte(`{"id":"module","name":"Roadmap","workspace_id":"workspace","project_id":"project"}`)
	got, err := moduleActivitySnapshotJSON(values, original)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id": "module", "workspace_id": "workspace", "project_id": "project", "name": "Roadmap", "description": "", "description_text": null, "description_html": null, "start_date": null, "target_date": null, "status": "planned", "lead_id": null, "member_ids": [], "view_props": {"layout": "list"}, "sort_order": 65535.0, "external_source": null, "external_id": null, "logo_props": {}, "total_estimate_points": 0.0, "completed_estimate_points": 0.0, "is_favorite": false, "total_issues": 0, "cancelled_issues": 0, "completed_issues": 0, "started_issues": 0, "unstarted_issues": 0, "backlog_issues": 0, "created_at": "2026-10-20T01:00:00-07:00", "updated_at": "2026-10-20T01:00:00-07:00", "archived_at": null}`
	if got != want {
		t.Fatalf("activity snapshot = %s\nwant %s", got, want)
	}
}

func TestValidateModuleWriteFieldsTracksDateOrderAndMemberReplacement(t *testing.T) {
	values, failures := validateModuleWriteFields(map[string]any{
		"name": "Roadmap", "start_date": "2026-11-02", "target_date": "2026-11-01",
		"member_ids": []any{}, "deleted_at": "2020-01-01T00:00:00Z", "archived_at": "2020-01-01T00:00:00Z",
	}, nil, false)
	if len(failures) != 0 {
		t.Fatalf("validation errors = %#v", failures)
	}
	if !reflect.DeepEqual(moduleDateOrderError(values), map[string]any{"non_field_errors": []string{"Start date cannot exceed target date"}}) {
		t.Fatalf("date object validation = %#v", moduleDateOrderError(values))
	}
	if members, ok := values["member_ids"].([]string); !ok || len(members) != 0 {
		t.Fatalf("empty member_ids = %#v, want an explicit empty replacement", values["member_ids"])
	}
	if _, writable := values["deleted_at"]; writable {
		t.Fatal("read-only deleted_at was accepted")
	}
	if _, writable := values["archived_at"]; writable {
		t.Fatal("read-only archived_at was accepted")
	}
}

func TestValidateModuleWriteFieldsDistinguishesOmittedMembers(t *testing.T) {
	values, failures := validateModuleWriteFields(map[string]any{"name": "Roadmap"}, nil, true)
	if len(failures) != 0 {
		t.Fatalf("validation errors = %#v", failures)
	}
	if _, present := values["member_ids"]; present {
		t.Fatal("omitted member_ids must preserve current membership")
	}
}

func TestModuleCreateDefaultsDoNotLeakIntoPutValidation(t *testing.T) {
	values, failures := validateModuleWriteFields(map[string]any{"name": "Renamed"}, nil, false)
	if len(failures) != 0 {
		t.Fatalf("PUT validation errors = %#v", failures)
	}
	for _, key := range []string{"status", "view_props", "logo_props", "sort_order"} {
		if _, present := values[key]; present {
			t.Errorf("PUT validation synthesized create default %q: %#v", key, values[key])
		}
	}
	applyModuleCreateDefaults(values)
	if values["status"] != "planned" || values["sort_order"] != float64(65535) {
		t.Fatalf("create defaults = %#v", values)
	}
	if !reflect.DeepEqual(values["view_props"], map[string]any{}) || !reflect.DeepEqual(values["logo_props"], map[string]any{}) {
		t.Fatalf("JSON create defaults = %#v", values)
	}
}

func TestModuleFieldErrorsSuppressObjectDateValidation(t *testing.T) {
	_, failures := validateModuleWriteFields(map[string]any{
		"name": nil, "start_date": "2026-11-02", "target_date": "2026-11-01",
	}, nil, true)
	if !reflect.DeepEqual(failures, map[string]any{"name": []string{"This field may not be null."}}) {
		t.Fatalf("validation errors = %#v", failures)
	}
}

type moduleMemberInsertTx struct {
	statements int
	pending    int
	persisted  int
	rollbacks  int
	commits    int
	failAt     int
}

func (tx *moduleMemberInsertTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	tx.statements++
	if tx.statements == tx.failAt {
		return pgconn.CommandTag{}, errors.New("injected insert failure")
	}
	tx.pending += 10
	return pgconn.NewCommandTag("INSERT 0 10"), nil
}

func (tx *moduleMemberInsertTx) Commit(context.Context) error {
	tx.commits++
	tx.persisted += tx.pending
	return nil
}

func (tx *moduleMemberInsertTx) Rollback(context.Context) error {
	tx.rollbacks++
	tx.pending = 0
	return nil
}

func TestModuleMemberReplacementInsertBatchesRollbackTogether(t *testing.T) {
	tx := &moduleMemberInsertTx{failAt: 2}
	members := make([]string, 11)
	for i := range members {
		members[i] = "member"
	}
	err := insertModuleMembersAtomically(context.Background(), func(context.Context) (moduleMemberTransaction, error) {
		return tx, nil
	}, "workspace", "project", "module", members, nil, nil, time.Unix(0, 0))
	if err == nil {
		t.Fatal("injected second batch failure unexpectedly succeeded")
	}
	if tx.statements != 2 || tx.commits != 0 || tx.rollbacks != 1 || tx.persisted != 0 {
		t.Fatalf("transaction statements=%d commits=%d rollbacks=%d persisted=%d", tx.statements, tx.commits, tx.rollbacks, tx.persisted)
	}
}

func TestValidateModuleAPIWriteFieldsIgnoresFieldsOutsideSerializer(t *testing.T) {
	values, failures := validateModuleAPIWriteFields(map[string]any{
		"name": "Roadmap", "lead_id": "f0530a31-b60e-4fca-850e-e23cb0ddfb92",
		"member_ids": []any{"f0530a31-b60e-4fca-850e-e23cb0ddfb92"}, "view_props": map[string]any{"layout": "list"},
		"deleted_at": "2026-01-01T00:00:00Z", "members": []any{},
	}, nil, false)
	if len(failures) != 0 {
		t.Fatalf("validation errors = %#v", failures)
	}
	for _, key := range []string{"lead_id", "deleted_at"} {
		if _, present := values[key]; present {
			t.Errorf("API serializer unexpectedly accepted app-only/read-only field %q", key)
		}
	}
	if members, ok := values["member_ids"].([]string); !ok || len(members) != 0 {
		t.Fatalf("API members = %#v, want explicit empty replacement", values["member_ids"])
	}
	if _, present := values["view_props"]; present {
		t.Fatalf("app-only view_props input affected API serializer validation: %#v", values["view_props"])
	}
}

func TestModuleMemberListFieldErrorsPreserveEveryIndex(t *testing.T) {
	values, failures := validateModuleWriteFields(map[string]any{
		"name":       "Roadmap",
		"member_ids": []any{nil, true, json.Number("42"), []any{"nested"}, map[string]any{"z": "first", "id": "not-a-uuid"}, ""},
	}, nil, false)
	want := map[string]any{"member_ids": map[string]any{
		"0": []string{"This field may not be null."},
		"1": []string{"Incorrect type. Expected pk value, received bool."},
		"3": []string{"Incorrect type. Expected pk value, received list."},
		"4": []string{"Incorrect type. Expected pk value, received dict."},
		"5": []string{"This field may not be null."},
	}}
	if !reflect.DeepEqual(failures, want) {
		t.Fatalf("member-list field errors = %#v, want %#v", failures, want)
	}
	if _, present := values["member_ids"]; present {
		t.Fatalf("partially converted member list escaped validation: %#v", values["member_ids"])
	}
}

func TestModuleMemberListFieldCollectsMultipleMalformedUUIDs(t *testing.T) {
	values, failures := validateModuleWriteFields(map[string]any{
		"name": "Roadmap", "member_ids": []any{"not-a-uuid", "also-not-a-uuid"},
	}, nil, false)
	want := map[string]any{"member_ids": map[string]any{
		"0": []string{"“not-a-uuid” is not a valid UUID."},
		"1": []string{"“also-not-a-uuid” is not a valid UUID."},
	}}
	if !reflect.DeepEqual(failures, want) {
		t.Fatalf("member-list field errors = %#v, want %#v", failures, want)
	}
	if _, present := values["member_ids"]; present {
		t.Fatalf("partially converted member list escaped validation: %#v", values["member_ids"])
	}
}

func TestModuleMemberCompositeValuesRejectWithoutTraversal(t *testing.T) {
	var deepList any = "leaf"
	var deepDict any = "leaf"
	for range 512 {
		deepList = []any{deepList}
		deepDict = map[string]any{"nested": deepDict}
	}
	fields := map[string]any{"name": "Roadmap", "member_ids": []any{deepList, deepDict, json.Number("42")}}
	var failures map[string]any
	allocations := testing.AllocsPerRun(20, func() {
		_, failures = validateModuleWriteFields(fields, nil, false)
	})
	want := map[string]any{"member_ids": map[string]any{
		"0": []string{"Incorrect type. Expected pk value, received list."},
		"1": []string{"Incorrect type. Expected pk value, received dict."},
	}}
	if !reflect.DeepEqual(failures, want) {
		t.Fatalf("composite member errors = %#v, want %#v", failures, want)
	}
	if allocations > 128 {
		t.Fatalf("deep composite validation used %.0f allocations; want at most 128 without rendering the nested value", allocations)
	}

	database := &moduleRelationDatabase{moduleDatabaseStub: &moduleDatabaseStub{rows: []pgx.Row{moduleBoolRow(false)}}}
	repository := &LifecycleRepository{database: database}
	relationErrors, err := repository.validateModuleRelations(context.Background(), fields, nil, "member_ids")
	if err != nil {
		t.Fatalf("composite relation validation: %v", err)
	}
	wantRelations := map[string]any{"member_ids": map[string]any{"2": []string{"Invalid pk \"42\" - object does not exist."}}}
	if !reflect.DeepEqual(relationErrors, wantRelations) {
		t.Fatalf("scalar relation validation = %#v, want %#v", relationErrors, wantRelations)
	}
	if len(database.queries) != 1 || len(database.args) != 1 || len(database.args[0]) != 1 {
		t.Fatalf("relation queries=%v args=%v, want only one lookup for the primitive numeric child", database.queries, database.args)
	}
}

type moduleRelationDatabase struct {
	*moduleDatabaseStub
}

func (*moduleRelationDatabase) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (*moduleRelationDatabase) Begin(context.Context) (pgx.Tx, error) {
	return nil, nil
}
