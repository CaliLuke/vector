package workitemtypes_test

import (
	"testing"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func TestPropertyGroupBucketsKeepContextAndEmptyDomainsSeparate(t *testing.T) {
	field := types.DiscoveryProperty{PropertyID: "location", Kind: property.Cascading, CatalogID: "current", LevelID: "country"}
	for _, sample := range []struct {
		name     string
		value    types.ProjectedValue
		expected string
	}{
		{"foreign error", types.ProjectedValue{Kind: property.Cascading, CatalogID: "other", Status: "error"}, "NotApplicable"},
		{"current error", types.ProjectedValue{Kind: property.Cascading, CatalogID: "current", Status: "error"}, "Error"},
		{"empty level", types.ProjectedValue{Kind: property.Cascading, CatalogID: "current", Status: "empty", Levels: map[string]any{"country": nil}}, "None"},
		{"missing level", types.ProjectedValue{Kind: property.Cascading, CatalogID: "current", Status: "value", Levels: map[string]any{"region": "id"}}, "NotApplicable"},
		{"current level", types.ProjectedValue{Kind: property.Cascading, CatalogID: "current", Status: "value", Levels: map[string]any{"country": "id"}}, "id"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			if got := types.PropertyGroupValue(field, map[string]types.ProjectedValue{"location": sample.value}); got != sample.expected {
				t.Fatalf("bucket=%q want %q", got, sample.expected)
			}
		})
	}
}
