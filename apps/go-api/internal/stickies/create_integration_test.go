//go:build integration

package stickies

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/go-plane/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestStickyCreatePersistsOwnedContentAndAllocation(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	pairURL := migratedDatabases.SeedDatabase(t, func(context.Context, string) error { return nil })

	seedNativeStickyDatabase(t, ctx, pairURL)
	pool := openStickyProofPool(t, ctx, pairURL)
	t.Cleanup(pool.Close)
	repository, err := NewRepository(pool)
	require.NoError(t, err)
	name, color, background, html, stripped := "Native note", "#5E6AD2", "#F3F4F6", "<p>Native note</p>", "Native note"
	created, err := repository.Create(ctx, StickyID(stickyProofSticky), Scope{
		Operation: OperationCreate, WorkspaceID: WorkspaceID(stickyProofWorkspace), UserID: UserID(stickyProofUser), Ownership: AssignOwner, AssignCreator: true,
	}, CreateStickyInput{Name: &name, Color: &color, BackgroundColor: &background, Description: []byte(`{"content":[],"type":"doc"}`), DescriptionHTML: &html, DescriptionStripped: &stripped, LogoProps: []byte(`{}`)})
	require.NoError(t, err)
	assertCreatedStickyIdentity(t, created)
	var storedName, storedHTML, storedText, owner, creator string
	var description []byte
	var sortOrder float64
	require.NoError(t, pool.QueryRow(ctx, `SELECT name,description,description_html,description_stripped,owner_id::text,created_by_id::text,sort_order FROM stickies WHERE id=$1 AND workspace_id=$2`, stickyProofSticky, stickyProofWorkspace).Scan(&storedName, &description, &storedHTML, &storedText, &owner, &creator, &sortOrder))
	require.Equal(t, name, storedName)
	require.Equal(t, html, storedHTML)
	require.Equal(t, stripped, storedText)
	require.Equal(t, stickyProofUser.String(), owner)
	require.Equal(t, owner, creator)
	require.Equal(t, float64(65535), sortOrder)
	require.JSONEq(t, `{"content":[],"type":"doc"}`, string(description))
}

func seedNativeStickyDatabase(t *testing.T, ctx context.Context, databaseURL string) {
	t.Helper()
	pool := openStickyProofPool(t, ctx, databaseURL)
	defer pool.Close()
	if _, err := pool.Exec(ctx, `INSERT INTO users (password,id,username,email,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ('!', $1, 'sticky-proof', 'sticky-proof@example.invalid', 'Sticky proof', '', '', '', TIMESTAMPTZ '2026-01-01 00:00:00+00', TIMESTAMPTZ '2026-01-01 00:00:00+00', TIMESTAMPTZ '2026-01-01 00:00:00+00', '', '', false, false, false, true, false, true, false, false, '', '', '', '', '', false, 'UTC', true)`, stickyProofUser); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (created_at,updated_at,id,name,owner_id,slug,timezone,background_color) VALUES (TIMESTAMPTZ '2026-01-01 00:00:00+00',TIMESTAMPTZ '2026-01-01 00:00:00+00',$1,'Sticky proof',$2,'test-workspace','UTC','#F3F4F6')`, stickyProofWorkspace, stickyProofUser); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspace_members (created_at,updated_at,id,workspace_id,member_id,role,view_props,default_props,issue_props,is_active,getting_started_checklist,tips,explored_features) VALUES (TIMESTAMPTZ '2026-01-01 00:00:00+00',TIMESTAMPTZ '2026-01-01 00:00:00+00',$1,$2,$3,20,'{}','{}','{}',true,'{}','{}','{}')`, stickyProofMembership, stickyProofWorkspace, stickyProofUser); err != nil {
		t.Fatal(err)
	}
}

func assertCreatedStickyIdentity(t *testing.T, sticky Sticky) {
	t.Helper()
	if uuid.UUID(sticky.ID) != stickyProofSticky || uuid.UUID(sticky.OwnerID) != stickyProofUser || uuid.UUID(sticky.WorkspaceID) != stickyProofWorkspace || sticky.CreatedByID == nil || uuid.UUID(*sticky.CreatedByID) != stickyProofUser || sticky.UpdatedByID != nil || sticky.SortOrder != 65535 {
		t.Fatalf("Go create returned unexpected identity or allocation: %#v", sticky)
	}
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
	stickyProofWorkspace  = uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290721")
	stickyProofMembership = uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290722")
	stickyProofUser       = uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290723")
	stickyProofSticky     = uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290724")
)
