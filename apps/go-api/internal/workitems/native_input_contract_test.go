package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/go-plane/internal/requestinput"
)

func TestIssueDatesAndIntegersUseNativeValidation(t *testing.T) {
	for _, test := range []struct {
		body, field string
		want        any
		invalid     bool
	}{
		{`{"start_date":"2024-02-29"}`, "start_date", "2024-02-29", false},
		{`{"target_date":null}`, "target_date", nil, false},
		{`{"start_date":"2026-1-2"}`, "start_date", nil, true},
		{`{"start_date":"0000-01-01"}`, "start_date", nil, true},
		{`{"target_date":"2026-02-29"}`, "target_date", nil, true},
		{`{"point":1.0}`, "point", int64(1), false},
		{`{"point":1e1}`, "point", int64(10), false},
		{`{"point":12}`, "point", int64(12), false},
		{`{"point":13}`, "point", nil, true},
		{`{"point":-1}`, "point", nil, true},
		{`{"point":"1.0"}`, "point", nil, true},
		{`{"point":"١"}`, "point", nil, true},
		{`{"point":true}`, "point", nil, true},
		{`{"sequence_id":"1_000"}`, "sequence_id", nil, true},
		{`{"sequence_id":1.5}`, "sequence_id", nil, true},
		{`{"sequence_id":2147483648}`, "sequence_id", nil, true},
		{`{"sequence_id":-2147483648}`, "sequence_id", int64(-2147483648), false},
		{`{"sequence_id":2147483647}`, "sequence_id", int64(2147483647), false},
	} {
		t.Run(test.body, func(t *testing.T) {
			input, err := requestinput.Parse(strings.NewReader(test.body), "application/json")
			if err != nil {
				t.Fatal(err)
			}
			changes, err := (&Repository{}).validateIssue(t.Context(), input, issueValidation{Partial: true})
			if test.invalid {
				var response *apicontract.HTTPError
				if !errors.As(err, &response) || response.Status != 400 || response.Body[test.field] == nil {
					t.Fatalf("invalid input: changes=%v error=%v", changes, err)
				}
			} else if err != nil || changes.values[test.field] != test.want {
				t.Fatalf("value=%#v error=%v want=%#v", changes.values[test.field], err, test.want)
			}
		})
	}
}

func TestIdentifierAndSearchRejectNonNativeIntegersBeforeDatabaseAccess(t *testing.T) {
	for _, api := range []bool{false, true} {
		for _, sequence := range []string{"١", "1_000", "1.0", "2147483648", strings.Repeat("9", 1001)} {
			_, err := (&Repository{}).identifier(context.Background(), Request{API: api, Params: map[string]string{"identifier": "TEST-" + sequence}})
			var response *apicontract.HTTPError
			if !errors.As(err, &response) || response.Status != 400 {
				t.Fatalf("API=%v sequence=%q error=%v", api, sequence, err)
			}
		}
	}
	for _, limit := range []string{"١", "1_0", "1.0", "1001", "-1", "9223372036854775808"} {
		_, err := (&Repository{}).search(t.Context(), Request{API: true, Params: map[string]string{"search": "needle", "limit": limit}, Present: map[string]bool{"limit": true}})
		var response *apicontract.HTTPError
		if !errors.As(err, &response) || response.Status != 400 {
			t.Fatalf("limit=%q error=%v", limit, err)
		}
	}
}

func TestSequenceSearchKeepsUnicodeWordBoundariesAndASCIISequences(t *testing.T) {
	got := searchSequences("TEST-0012 (34) ١٢ x56 78x 90_ é123 ٤56")
	if !reflect.DeepEqual(got, []string{"12", "34"}) {
		t.Fatalf("sequences=%v", got)
	}
}

func TestBulkDatePreparationRejectsWholeBatchAndKeepsSourceSnapshots(t *testing.T) {
	const first = "00000000-0000-4000-8000-000000000001"
	const second = "00000000-0000-4000-8000-000000000002"
	rows := map[string]map[string]any{
		first:  {"start_date": "2026-01-01", "target_date": "2026-01-10"},
		second: {"start_date": nil, "target_date": nil},
	}
	for _, invalid := range []any{"2026-1-2", "2026-02-30", true, json.Number("1")} {
		updates := []any{map[string]any{"id": first, "start_date": "2026-01-02"}, map[string]any{"id": second, "start_date": invalid}}
		changed, activities, err := prepareBulkDateChanges(updates, rows)
		var response *apicontract.HTTPError
		if !errors.As(err, &response) || response.Status != 400 || changed != nil || activities != nil || rows[first]["start_date"] != "2026-01-01" {
			t.Fatalf("invalid=%v changed=%v activities=%v error=%v source=%v", invalid, changed, activities, err, rows)
		}
	}
	updates := []any{
		map[string]any{"id": first, "start_date": "2026-01-02"},
		map[string]any{"id": first, "start_date": nil},
		map[string]any{"id": "00000000-0000-4000-8000-000000000099", "target_date": "2026-01-03"},
	}
	changed, activities, err := prepareBulkDateChanges(updates, rows)
	if err != nil || len(changed) != 1 || len(activities) != 2 || changed[first]["start_date"] != nil || activities[1].previous != "2026-01-02" || rows[first]["start_date"] != "2026-01-01" {
		t.Fatalf("changes=%v activities=%v error=%v source=%v", changed, activities, err, rows)
	}
}
