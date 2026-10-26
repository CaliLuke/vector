package issuequery

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const selectedTypeA = "00000000-0000-4000-8000-000000000001"
const selectedTypeB = "00000000-0000-4000-8000-000000000002"

func TestTypeFilterRequiresExplicitAuthorizedCatalog(t *testing.T) {
	for _, raw := range []string{`{"type_id":"` + selectedTypeA + `"}`, `{"not":{"type_id__in":"` + selectedTypeA + `"}}`} {
		args := []any{}
		_, err := BuildComplexFilter(raw, &args)
		var invalid *FilterError
		if !errors.As(err, &invalid) || invalid.Code != "invalid_filter_field" {
			t.Fatalf("public type filter accepted: %v", err)
		}
	}
}

func TestTypeFilterBindsAllSelectedIDsWithoutLegacyArrayTruncation(t *testing.T) {
	catalog := map[string]bool{selectedTypeA: true, selectedTypeB: true}
	args := []any{"actor"}
	sql, err := BuildComplexFilterWithTypes(`{"type_id__in":["`+selectedTypeA+`","`+selectedTypeB+`"]}`, &args, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if sql != "COALESCE(issue.type_id = ANY($2::uuid[]),FALSE)" || !reflect.DeepEqual(args, []any{"actor", []string{selectedTypeA, selectedTypeB}}) {
		t.Fatalf("predicate=%s args=%v", sql, args)
	}
}

func TestTypeFilterUnknownSelectionIsRepairableRatherThanEmpty(t *testing.T) {
	args := []any{}
	_, err := BuildComplexFilterWithTypes(`{"type_id__in":"`+selectedTypeA+`,`+selectedTypeB+`"}`, &args, map[string]bool{selectedTypeA: true})
	var invalid *FilterError
	if !errors.As(err, &invalid) || invalid.Code != "invalid_filter_reference" {
		t.Fatalf("missing selection did not require repair: %v", err)
	}
	if len(args) != 0 {
		t.Fatalf("rejected filter bound args: %v", args)
	}
}

func TestTypeFilterRejectsMalformedSelectionAndSingleEqualityArrays(t *testing.T) {
	for _, raw := range []string{`{"type_id__exact":["` + selectedTypeA + `"]}`, `{"type_id__in":[null]}`, `{"type_id__in":42}`, `{"type_id__in":"invalid"}`} {
		args := []any{}
		if _, err := BuildComplexFilterWithTypes(raw, &args, map[string]bool{selectedTypeA: true}); err == nil {
			t.Fatalf("accepted invalid selection: %s", raw)
		}
	}
}

func TestTypeFilterNegationIncludesUntypedItemsAndEmptySelectionStaysFalse(t *testing.T) {
	args := []any{}
	sql, err := BuildComplexFilterWithTypes(`{"not":{"type_id__in":"`+selectedTypeA+`"}}`, &args, map[string]bool{selectedTypeA: true})
	if err != nil || !strings.Contains(sql, ") IS NOT TRUE") || !strings.Contains(sql, "COALESCE(issue.type_id = ANY($1::uuid[]),FALSE)") {
		t.Fatalf("negation lost null policy: %s %v", sql, err)
	}
	sql, err = BuildComplexFilterWithTypes(`{"type_id__in":[]}`, &args, map[string]bool{})
	if err != nil || sql != "FALSE" {
		t.Fatalf("empty inclusion broadened results: %s %v", sql, err)
	}
	sql, err = BuildComplexFilterWithTypes(`{"type_id__exact":null}`, &args, map[string]bool{})
	if err != nil || sql != "issue.type_id IS NULL" {
		t.Fatalf("untyped selection: %s %v", sql, err)
	}
}

func TestTypeFilterLogicalCasingCannotBroadenSavedSelections(t *testing.T) {
	args := []any{}
	sql, err := BuildComplexFilterWithTypes(`{"nOt":{"OR":[{"type_id__exact":"`+selectedTypeA+`"},{"type_id__exact":"`+selectedTypeB+`"}]}}`, &args, map[string]bool{selectedTypeA: true, selectedTypeB: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 2 || !strings.Contains(sql, " OR ") || !strings.HasSuffix(sql, ") IS NOT TRUE") {
		t.Fatalf("logical selection was lost: %s %v", sql, args)
	}
	legacyArgs := []any{}
	sql, err = BuildComplexFilter(`{"NOT":{"priority__exact":"high"}}`, &legacyArgs)
	if err != nil || len(legacyArgs) != 1 || !strings.HasSuffix(sql, ") IS NOT TRUE") {
		t.Fatalf("accepted legacy casing became no filter: %s %v", sql, err)
	}
}

func TestTypeFilterDetectionUsesParsedKeysAndLogicalStructure(t *testing.T) {
	for _, raw := range []string{`{"type_id":null}`, `{"nOt":{"OR":[{"priority":"high"},{"type_id__in":[]}]}}`, `{"type_\u0069d__exact":null}`} {
		if !UsesTypeFilters(raw) {
			t.Fatalf("missed type filter: %s", raw)
		}
	}
	for _, raw := range []string{"", `{"name__exact":"type_id"}`, `{"priority":{"type_id":null}}`, `{"type_id":null} trailing`, `{`} {
		if UsesTypeFilters(raw) {
			t.Fatalf("selected snapshot from non-filter: %s", raw)
		}
	}
}
