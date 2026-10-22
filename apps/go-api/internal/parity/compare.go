package parity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/google/go-cmp/cmp"
)

// RuntimeTransition is one runtime's complete application-table state change.
type RuntimeTransition struct {
	Before                Snapshot
	After                 Snapshot
	Response              CapturedResponse
	ExpectedChangedTables []string
	ChangedTables         []string
}

// ComparisonProjection contains only safe table locations, counts, and row fingerprints.
type ComparisonProjection struct {
	Tables []TableProjection
}

// TableProjection is a safe representation of one complete ordered table dump.
type TableProjection struct {
	Name        string
	Columns     []Column
	PrimaryKey  []string
	RowCount    int
	Fingerprint []string
}

// MismatchError contains the standard go-cmp diff of safe typed projections.
type MismatchError struct {
	Diff string
}

func (e *MismatchError) Error() string {
	return "parity failed with unexplained differences"
}

// NewRuntimeTransition derives and independently verifies one runtime's changed-table set.
func NewRuntimeTransition(
	before Snapshot,
	after Snapshot,
	response CapturedResponse,
	expectedChangedTables []string,
) (RuntimeTransition, error) {
	changedTables, err := deriveChangedTables(before, after)
	if err != nil {
		return RuntimeTransition{}, err
	}
	expected := sortedUnique(expectedChangedTables)
	if diff := cmp.Diff(expected, changedTables); diff != "" {
		return RuntimeTransition{}, fmt.Errorf("unexpected changed tables (-expected +observed):\n%s", diff)
	}
	return RuntimeTransition{
		Before:                before,
		After:                 after,
		Response:              response,
		ExpectedChangedTables: expected,
		ChangedTables:         changedTables,
	}, nil
}

// DeclaredNormalizationsOption returns the named go-cmp projection used for all snapshot comparison.
// It applies only reviewed top-level table/column rules, then hashes row contents before cmp.Diff can report them.
func DeclaredNormalizationsOption(normalizations Normalizations) cmp.Option {
	return cmp.Transformer(normalizationOptionName(normalizations), func(snapshot Snapshot) ComparisonProjection {
		return projectSnapshot(snapshot, normalizations)
	})
}

func indexTables(tables []TableDump) map[string]TableDump {
	indexed := make(map[string]TableDump, len(tables))
	for _, table := range tables {
		indexed[table.Name] = table
	}
	return indexed
}

func unionTableNames(left map[string]TableDump, right map[string]TableDump) []string {
	names := make([]string, 0, len(left)+len(right))
	seen := make(map[string]struct{}, len(left)+len(right))
	for name := range left {
		seen[name] = struct{}{}
	}
	for name := range right {
		seen[name] = struct{}{}
	}
	for name := range seen {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// normalizationOptionName makes every reviewed rule identity and exact rule-set digest visible to go-cmp.
// Separate Snapshot transformers per rule would be ambiguous to go-cmp because they share the same types.
func normalizationOptionName(normalizations Normalizations) string {
	parts := []string{"DeclaredNormalizations"}
	for _, rule := range normalizations.VolatileFields {
		parts = append(parts, "Field", rule.Table, rule.Column)
	}
	for _, rule := range normalizations.VolatileValues {
		parts = append(parts, "Values", rule.Table, rule.Column)
	}
	encoded, _ := json.Marshal(normalizations)
	digest := sha256.Sum256(encoded)
	parts = append(parts, hex.EncodeToString(digest[:6]))
	name := strings.Join(parts, "_")
	return strings.Map(func(character rune) rune {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || character == '_' {
			return character
		}
		return '_'
	}, name)
}

// CompareTransitions uses cmp.Diff on safe projections and fails on every unexplained difference.
func CompareTransitions(source RuntimeTransition, target RuntimeTransition, options ...cmp.Option) (string, error) {
	comparisonOptions := append([]cmp.Option{SafeResponseOption()}, options...)
	diff := cmp.Diff(source, target, comparisonOptions...)
	if diff != "" {
		return diff, &MismatchError{Diff: diff}
	}
	return "", nil
}

func deriveChangedTables(before Snapshot, after Snapshot) ([]string, error) {
	beforeTables := indexTables(before.Tables)
	afterTables := indexTables(after.Tables)
	tableNames := unionTableNames(beforeTables, afterTables)
	changed := make([]string, 0)
	for _, tableName := range tableNames {
		beforeTable, beforeExists := beforeTables[tableName]
		afterTable, afterExists := afterTables[tableName]
		if !beforeExists || !afterExists {
			return nil, fmt.Errorf("complete dumps do not contain the same table %q", tableName)
		}
		if diff := cmp.Diff(beforeTable, afterTable); diff != "" {
			changed = append(changed, tableName)
		}
	}
	return changed, nil
}

func projectSnapshot(snapshot Snapshot, normalizations Normalizations) ComparisonProjection {
	projection := ComparisonProjection{Tables: make([]TableProjection, 0, len(snapshot.Tables))}
	for _, table := range snapshot.Tables {
		projectedTable := TableProjection{
			Name:        table.Name,
			Columns:     table.Columns,
			PrimaryKey:  table.PrimaryKey,
			RowCount:    len(table.Rows),
			Fingerprint: make([]string, 0, len(table.Rows)),
		}
		for rowIndex, row := range table.Rows {
			canonical, err := normalizations.canonicalize(table.Name, row)
			if err != nil {
				panic(fmt.Sprintf("canonicalize %s row %d: %v", table.Name, rowIndex, err))
			}
			projectedTable.Fingerprint = append(projectedTable.Fingerprint, fingerprint(canonical))
		}
		projection.Tables = append(projection.Tables, projectedTable)
	}
	return projection
}
