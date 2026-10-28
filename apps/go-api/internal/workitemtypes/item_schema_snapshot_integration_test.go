//go:build integration

package workitemtypes_test

import (
	"context"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func TestItemSchemaSnapshotOnlyReturnsActiveScopedSavedValues(t *testing.T) {
	f := newProjectTypeFixture(t)
	effort, flag, hidden, computed, absent := uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: effort, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true, Default: float64(7)}},
		property.Definition{ID: flag, Title: "Verified", Kind: property.Boolean, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: hidden, Title: "Hidden", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Single line"}, Contexts: []property.Context{{ID: uuid.New().String(), Name: "Hidden here", Projects: []string{f.project}, Overrides: []string{"active"}, Configuration: property.Configuration{Active: false}}}},
		property.Definition{ID: computed, Title: "Calculation", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{effort}} + 1", Result: "Number"}},
		property.Definition{ID: absent, Title: "Optional", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Single line"}},
	)
	repo, draft, _ := seedPublicationDraft(t, f)
	result, err := publishDraftForTest(f, repo, draft, f.member, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	issue := resultIssueID(t, result)
	foreign := uuid.New().String()
	updated := time.Date(2026, 10, 25, 12, 34, 56, 0, time.UTC)
	f.exec(`UPDATE issues SET updated_at=$2,custom_properties=jsonb_build_object($3::text,0,$4::text,false,$5::text,'retained secret',$6::text,900,$7::text,'foreign secret') WHERE id=$1`, issue, updated, effort, flag, hidden, computed, foreign)
	input := types.ItemSchemaInput{Mode: "edit", ProjectID: f.project, IssueID: issue}
	schema, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(schema.StoredValues) != 2 || schema.StoredValues[effort] != float64(0) || schema.StoredValues[flag] != false || schema.Computed[computed].Value != float64(1) || schema.ItemUpdatedAt == nil || !schema.ItemUpdatedAt.Equal(updated) {
		t.Fatalf("incoherent saved snapshot: %+v", schema)
	}
	// Preview computes with unsaved inputs without relabelling them as saved values.
	input.Preview = &types.ItemFormulaPreview{Values: map[string]any{effort: float64(4)}}
	preview, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || preview.StoredValues[effort] != float64(0) || preview.Computed[computed].Value != float64(5) {
		t.Fatalf("preview changed stored values: %+v %v", preview, err)
	}
	f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
	input.Preview = nil
	disabled, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || len(disabled.StoredValues) != 0 {
		t.Fatalf("disabled snapshot exposed values: %+v %v", disabled, err)
	}
}

// Run a real second-connection write between the saved-row read and formula reads.
// The fixture deliberately interleaves the transactions without timing sleeps.
type interleavedSchemaDatabase struct {
	db        *pgx.Conn
	afterRead func() error
}

func (d interleavedSchemaDatabase) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := d.db.Begin(ctx)
	return interleavedSchemaTransaction{Tx: tx, afterRead: d.afterRead}, err
}

type interleavedSchemaTransaction struct {
	pgx.Tx
	afterRead func() error
}

func (tx interleavedSchemaTransaction) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	row := tx.Tx.QueryRow(ctx, sql, args...)
	if strings.Contains(sql, "i.custom_properties,i.updated_at FROM issues") {
		return interleavedSchemaRow{Row: row, afterRead: tx.afterRead}
	}
	return row
}

type interleavedSchemaRow struct {
	pgx.Row
	afterRead func() error
}

func (r interleavedSchemaRow) Scan(dest ...any) error {
	if err := r.Row.Scan(dest...); err != nil {
		return err
	}
	return r.afterRead()
}
func TestItemSchemaSnapshotDoesNotMixConcurrentSavedVersions(t *testing.T) {
	f := newProjectTypeFixture(t)
	effort, calculation := uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: effort, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: calculation, Title: "Calculation", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{effort}} + LEN({{name}})", Result: "Number"}},
	)
	repo, draft, _ := seedPublicationDraft(t, f)
	result, err := publishDraftForTest(f, repo, draft, f.member, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	issue := resultIssueID(t, result)
	first := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	next := first.Add(time.Minute)
	f.exec(`UPDATE issues SET name='a',custom_properties=jsonb_build_object($2::text,0),updated_at=$3 WHERE id=$1`, issue, effort, first)
	writer, err := pgx.Connect(f.ctx, f.db.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.CloseContext(t, writer, context.WithoutCancel(f.ctx)) }()
	wrote := false
	interleaved := types.NewRepository(interleavedSchemaDatabase{db: f.db, afterRead: func() error {
		_, writeErr := writer.Exec(f.ctx, `UPDATE issues SET name='changed',custom_properties=jsonb_build_object($2::text,9),updated_at=$3 WHERE id=$1`, issue, effort, next)
		wrote = writeErr == nil
		return writeErr
	}})
	input := types.ItemSchemaInput{Mode: "edit", ProjectID: f.project, IssueID: issue}
	before, err := interleaved.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil {
		t.Fatal(err)
	}
	if !wrote || before.StoredValues[effort] != float64(0) || before.Computed[calculation].Value != float64(1) || before.ItemUpdatedAt == nil || !before.ItemUpdatedAt.Equal(first) {
		t.Fatalf("mixed versions: %+v", before)
	}
	after, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || after.StoredValues[effort] != float64(9) || after.Computed[calculation].Value != float64(16) || after.ItemUpdatedAt == nil || !after.ItemUpdatedAt.Equal(next) {
		t.Fatalf("new snapshot: %+v %v", after, err)
	}
}
