//go:build integration

package workitemtypes_test

import (
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestDraftDeletionRollsBackWhenDurableAdmissionFails(t *testing.T) {
	f := newProjectTypeFixture(t)
	repository, draft, _ := seedPublicationDraft(t, f)
	f.exec(`CREATE FUNCTION reject_draft_cleanup() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'draft cleanup unavailable'; END $$;
CREATE TRIGGER reject_draft_cleanup BEFORE INSERT ON work_item_effects FOR EACH ROW EXECUTE FUNCTION reject_draft_cleanup()`)

	_, err := repository.Execute(f.ctx, workitems.Request{
		Operation: "api_workspaces_by_slug_draft_issues_by_pk", Method: "DELETE",
		ActorID: f.member, Slug: "types", Timezone: "UTC", Params: map[string]string{"pk": draft},
	}, nil, "application/json")
	var rejected *pgconn.PgError
	if !errors.As(err, &rejected) || rejected.Code != "P0001" {
		t.Fatalf("expected cleanup admission rejection, got %v", err)
	}
	var live bool
	var effects int
	err = f.db.QueryRow(f.ctx, `SELECT deleted_at IS NULL,(SELECT count(*) FROM work_item_effects) FROM draft_issues WHERE id=$1`, draft).Scan(&live, &effects)
	if err != nil || !live || effects != 0 {
		t.Fatalf("failed deletion changed draft or outbox: live=%v effects=%d error=%v", live, effects, err)
	}
}
