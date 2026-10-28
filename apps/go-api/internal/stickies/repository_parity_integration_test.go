//go:build integration

package stickies

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/parity"
	"github.com/jackc/pgx/v5/pgxpool"
)

type recordedCreateFixture struct {
	Response parity.CapturedResponse `json:"response"`
}

// TestStickyCreateDifferentialTransition compares live complete database
// transitions. The source action applies the recorded Django row contract. The
// target action uses the real Go repository.
func TestStickyCreateDifferentialTransition(t *testing.T) {
	if os.Getenv("PLANE_RUN_STICKY_DIFFERENTIAL_PROOF") != "1" {
		t.Skip("set PLANE_RUN_STICKY_DIFFERENTIAL_PROOF=1 to run the Sticky differential proof")
	}
	sourceURL, targetURL := os.Getenv("PLANE_STICKY_SOURCE_POSTGRES_URL"), os.Getenv("PLANE_STICKY_TARGET_POSTGRES_URL")
	if sourceURL == "" || targetURL == "" {
		t.Skip("set distinct migrated PLANE_STICKY_SOURCE_POSTGRES_URL and PLANE_STICKY_TARGET_POSTGRES_URL values")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	seedDifferentialDatabase(t, ctx, sourceURL)
	seedDifferentialDatabase(t, ctx, targetURL)
	fixture := loadRecordedCreateFixture(t)

	diff, err := parity.CaptureAndCompare(ctx, parity.Config{
		SourcePostgresURL: sourceURL, TargetPostgresURL: targetURL, Schema: "public",
		ExpectedChangedTables: []string{"stickies"},
		Normalizations: parity.Normalizations{VolatileFields: []parity.VolatileFieldRule{
			{Table: "stickies", Column: "created_at"}, {Table: "stickies", Column: "updated_at"},
		}},
	}, func(ctx context.Context) (parity.ActionResult, error) {
		source := openStickyProofPool(t, ctx, sourceURL)
		defer source.Close()
		_, err := source.Exec(ctx, `INSERT INTO stickies (created_at,updated_at,id,name,description,description_html,description_stripped,logo_props,color,background_color,created_by_id,owner_id,workspace_id,sort_order) VALUES (CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,$1,'Migration fixture','{"content":[],"type":"doc"}','<p>Migration fixture</p>','Migration fixture','{}','#5E6AD2','#F3F4F6',$2,$2,$3,65535)`, differentialSticky, differentialUser, differentialWorkspace)
		if err != nil {
			return parity.ActionResult{}, err
		}
		target := openStickyProofPool(t, ctx, targetURL)
		defer target.Close()
		repository, err := NewRepository(target)
		if err != nil {
			return parity.ActionResult{}, err
		}
		name, color, background, html, stripped := "Migration fixture", "#5E6AD2", "#F3F4F6", "<p>Migration fixture</p>", "Migration fixture"
		created, err := repository.Create(ctx, StickyID(differentialSticky), Scope{
			Operation: OperationCreate, WorkspaceID: WorkspaceID(differentialWorkspace), UserID: UserID(differentialUser), Ownership: AssignOwner, AssignCreator: true,
		}, CreateStickyInput{
			Name: &name, Color: &color, BackgroundColor: &background,
			Description: []byte(`{"content":[],"type":"doc"}`), DescriptionHTML: &html,
			DescriptionStripped: &stripped, LogoProps: []byte(`{}`),
		})
		if err != nil {
			return parity.ActionResult{}, err
		}
		assertRecordedCreateResult(t, created)
		assertJSONEquivalent(t, canonicalCreatedResponse(t, created), fixture.Response.Body)
		return parity.ActionResult{SourceResponse: fixture.Response, TargetResponse: fixture.Response}, nil
	})
	if err != nil || diff != "" {
		t.Fatalf("recorded Django versus live Go transition error = %v, diff = %q", err, diff)
	}
}

func seedDifferentialDatabase(t *testing.T, ctx context.Context, databaseURL string) {
	t.Helper()
	pool := openStickyProofPool(t, ctx, databaseURL)
	defer pool.Close()
	if _, err := pool.Exec(ctx, `INSERT INTO users (password,id,username,email,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ('!', $1, 'sticky-proof', 'sticky-proof@example.invalid', 'Sticky proof', '', '', '', TIMESTAMPTZ '2026-01-01 00:00:00+00', TIMESTAMPTZ '2026-01-01 00:00:00+00', TIMESTAMPTZ '2026-01-01 00:00:00+00', '', '', false, false, false, true, false, true, false, false, '', '', '', '', '', false, 'UTC', true)`, differentialUser); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (created_at,updated_at,id,name,owner_id,slug,timezone,background_color) VALUES (TIMESTAMPTZ '2026-01-01 00:00:00+00',TIMESTAMPTZ '2026-01-01 00:00:00+00',$1,'Sticky proof',$2,'test-workspace','UTC','#F3F4F6')`, differentialWorkspace, differentialUser); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspace_members (created_at,updated_at,id,workspace_id,member_id,role,view_props,default_props,issue_props,is_active,getting_started_checklist,tips,explored_features) VALUES (TIMESTAMPTZ '2026-01-01 00:00:00+00',TIMESTAMPTZ '2026-01-01 00:00:00+00',$1,$2,$3,20,'{}','{}','{}',true,'{}','{}','{}')`, differentialMembership, differentialWorkspace, differentialUser); err != nil {
		t.Fatal(err)
	}
}

func loadRecordedCreateFixture(t *testing.T) recordedCreateFixture {
	t.Helper()
	data, err := os.ReadFile("../parity/testdata/fixtures/stickies/create.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture recordedCreateFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func assertRecordedCreateResult(t *testing.T, sticky Sticky) {
	t.Helper()
	if uuid.UUID(sticky.ID) != differentialSticky || uuid.UUID(sticky.OwnerID) != differentialUser || uuid.UUID(sticky.WorkspaceID) != differentialWorkspace || sticky.CreatedByID == nil || uuid.UUID(*sticky.CreatedByID) != differentialUser || sticky.UpdatedByID != nil || sticky.SortOrder != 65535 {
		t.Fatalf("Go create returned unexpected identity or allocation: %#v", sticky)
	}
}

func canonicalCreatedResponse(t *testing.T, sticky Sticky) json.RawMessage {
	t.Helper()
	return mustJSON(t, map[string]any{
		"background_color": *sticky.BackgroundColor, "color": *sticky.Color,
		"created_at": "<timestamp>", "created_by": "<uuid-1>", "deleted_at": nil,
		"description": map[string]any{"content": []any{}, "type": "doc"}, "description_binary": nil,
		"description_html": sticky.DescriptionHTML, "description_stripped": *sticky.DescriptionStripped,
		"id": "<uuid-2>", "logo_props": map[string]any{}, "name": *sticky.Name, "owner": "<uuid-1>",
		"sort_order": sticky.SortOrder, "updated_at": "<timestamp>", "updated_by": nil, "workspace": "<uuid-3>",
	})
}

func assertJSONEquivalent(t *testing.T, actual, expected json.RawMessage) {
	t.Helper()
	var actualValue, expectedValue any
	if err := json.Unmarshal(actual, &actualValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(expected, &expectedValue); err != nil {
		t.Fatal(err)
	}
	actualCanonical, expectedCanonical := mustJSON(t, actualValue), mustJSON(t, expectedValue)
	if string(actualCanonical) != string(expectedCanonical) {
		t.Fatalf("Go response does not match the recorded Django body: got %s, want %s", actualCanonical, expectedCanonical)
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func openStickyProofPool(t *testing.T, ctx context.Context, databaseURL string) *pgxpool.Pool {
	t.Helper()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if configurePoolErr := database.ConfigurePool(config); configurePoolErr != nil {
		t.Fatal(configurePoolErr)
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	return pool
}

var (
	differentialWorkspace  = uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290721")
	differentialMembership = uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290722")
	differentialUser       = uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290723")
	differentialSticky     = uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290724")
)
