package issuequery

import (
	"reflect"
	"strings"
	"testing"
)

func acceptTestProperty(field string, value any) (bool, error) {
	return strings.HasPrefix(field, "property__"), nil
}

func TestPropertyFilterRequiresAuthorizedValidatorAndCompleteEvaluation(t *testing.T) {
	raw := `{"property__effort__exact":4}`
	if _, err := PreparePropertyFilter(raw, nil, nil); err == nil {
		t.Fatal("unrecognized property accepted")
	}
	plan, err := PreparePropertyFilter(raw, nil, acceptTestProperty)
	if err != nil {
		t.Fatal(err)
	}
	for _, matches := range []map[int]PropertyFilterMatches{
		nil,
		{2: {}},
		{1: {TrueIDs: []string{selectedTypeA}, FalseIDs: []string{selectedTypeA}}},
		{1: {TrueIDs: []string{"not an item"}}},
	} {
		if _, err = plan.Bind(matches); err == nil {
			t.Fatalf("invalid materialization accepted: %+v", matches)
		}
	}
}

func TestPropertyFilterCapturesLeavesAndCopiesMaterialization(t *testing.T) {
	plan, err := PreparePropertyFilter(`{"and":[{"property__effort__in":[0,4]},{"not":{"property__effort__in":[4,8]}}]}`, nil, acceptTestProperty)
	if err != nil {
		t.Fatal(err)
	}
	leaves := plan.Leaves()
	if len(leaves) != 2 || leaves[0].ID == leaves[1].ID {
		t.Fatalf("occurrences lost: %+v", leaves)
	}
	leaves[0].Value.([]any)[0] = "changed"
	if plan.Leaves()[0].Value.([]any)[0] == "changed" {
		t.Fatal("leaf mutation escaped")
	}
	source := []string{selectedTypeA, selectedTypeA}
	bound, err := plan.Bind(map[int]PropertyFilterMatches{1: {TrueIDs: source}, 2: {FalseIDs: []string{selectedTypeB}}})
	if err != nil {
		t.Fatal(err)
	}
	source[0] = selectedTypeB
	args := []any{}
	if _, err = bound.SQL(&args); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(args[0], map[string]bool{selectedTypeA: true}) {
		t.Fatalf("caller mutation or duplicates retained: %+v", args)
	}
}

func TestPreparedPropertyFilterRetainsLegacyValidationAndSQL(t *testing.T) {
	for _, raw := range []string{"", `null`, `false`, `0`, `[]`, `{}`, `{"priority__exact":"high"}`, `{"not":{"label_id__exact":"` + selectedTypeA + `"}}`} {
		plan, err := PreparePropertyFilter(raw, nil, acceptTestProperty)
		if err != nil {
			t.Fatal(err)
		}
		bound, err := plan.Bind(nil)
		if err != nil {
			t.Fatal(err)
		}
		leftArgs, rightArgs := []any{}, []any{}
		left, err := bound.SQL(&leftArgs)
		if err != nil {
			t.Fatal(err)
		}
		right, err := BuildComplexFilter(raw, &rightArgs)
		if err != nil || left != right || !reflect.DeepEqual(leftArgs, rightArgs) {
			t.Fatalf("legacy drift %s: %s / %s %v", raw, left, right, err)
		}
	}
	for _, raw := range []string{`{"or":[]}`, `{"not":{"property__effort":{}}}`, `{"property__effort":[{}]}`, `{"and":[{"property__effort":4}],"priority":"high"}`, `{"property__effort":4} trailing`} {
		if _, err := PreparePropertyFilter(raw, nil, acceptTestProperty); err == nil {
			t.Fatalf("invalid expression accepted: %s", raw)
		}
	}
}
