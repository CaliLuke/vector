//go:build integration

package workitemhistory

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func testRetainedPropertyReads(t *testing.T, ctx context.Context, db *pgx.Conn, request Request) {
	t.Run("retained property revisions and visibility", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { testsupport.Rollback(t, tx, ctx) }()
		_, err = tx.Exec(ctx, `CREATE TEMP TABLE work_item_property_versions(id uuid PRIMARY KEY,workspace_id uuid NOT NULL,issue_id uuid NOT NULL,property_id uuid NOT NULL,value jsonb NOT NULL,actor_id uuid,created_at timestamptz NOT NULL,property_title text NOT NULL,source text NOT NULL)`)
		if err != nil {
			t.Fatal(err)
		}
		field := uuid.New()
		_, err = tx.Exec(ctx, `INSERT INTO work_item_property_versions SELECT gen_random_uuid(),$1,$2,$3,'"<p>Saved text</p>"'::jsonb,$4,now()-n*interval '1 minute','Retired notes','change' FROM generate_series(1,52)n`, requestWorkspace(), request.ResourceID, field, request.UserID)
		if err != nil {
			t.Fatal(err)
		}
		repository, err := NewRepository(tx)
		if err != nil {
			t.Fatal(err)
		}
		request.PropertyID = field
		request.Operation = "property_history"
		raw, err := repository.Read(ctx, request)
		if err != nil {
			t.Fatal(err)
		}
		var histories []struct {
			Title string `json:"title"`
			Count int    `json:"version_count"`
		}
		if err = json.Unmarshal(raw, &histories); err != nil {
			t.Fatal(err)
		}
		if len(histories) != 1 || histories[0].Title != "Retired notes" || histories[0].Count != 52 || strings.Contains(string(raw), "Saved text") {
			t.Fatalf("retained history metadata=%s", raw)
		}
		request.Operation = "property_versions"
		raw, err = repository.Read(ctx, request)
		if err != nil {
			t.Fatal(err)
		}
		var page struct {
			Results []map[string]any `json:"results"`
			Next    *uuid.UUID       `json:"next_cursor"`
		}
		if err = json.Unmarshal(raw, &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Results) != 50 || page.Next == nil || strings.Contains(string(raw), "Saved text") {
			t.Fatalf("first revision page=%s", raw)
		}
		current := uuid.New()
		_, err = tx.Exec(ctx, `INSERT INTO work_item_property_versions VALUES($1,$2,$3,$4,to_jsonb($5::text),NULL,now(),'Retired notes','change')`, current, requestWorkspace(), request.ResourceID, field, `<p>New text</p><script>alert(1)</script>`)
		if err != nil {
			t.Fatal(err)
		}
		request.BeforeID = page.Next
		raw, err = repository.Read(ctx, request)
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(raw, &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Results) != 2 || page.Next != nil {
			t.Fatalf("cursor shifted after append: %s", raw)
		}
		request.Operation, request.VersionID = "property_version_detail", current
		raw, err = repository.Read(ctx, request)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "New text") || strings.Contains(string(raw), "alert") || strings.Contains(string(raw), "script") {
			t.Fatalf("unsafe revision detail=%s", raw)
		}
		request.PropertyID = uuid.New()
		if _, err = repository.Read(ctx, request); !errors.Is(err, ErrNotFound) {
			t.Fatalf("foreign field version=%v", err)
		}
		request.PropertyID = field
		_, err = tx.Exec(ctx, `UPDATE workspace_members SET is_active=false`)
		if err != nil {
			t.Fatal(err)
		}
		for _, operation := range []string{"property_history", "property_versions", "property_version_detail"} {
			request.Operation = operation
			raw, err = repository.Read(ctx, request)
			if !errors.Is(err, ErrNotFound) || raw != nil {
				t.Fatalf("revoked %s exposed %s (%v)", operation, raw, err)
			}
		}
	})
}

func requestWorkspace() uuid.UUID { return uuid.MustParse(historyWorkspace) }
