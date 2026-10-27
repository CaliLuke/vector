//go:build integration

package milestones_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMilestoneRetainedArchivedIntakeLinksRemainRemovable(t *testing.T) {
	f := newWriteFixture(t)
	created := f.milestone(t, "Release")
	id, item, intake := uuid.MustParse(created.MilestoneID), f.item(t), uuid.New()
	_, err := f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{item}, nil)
	require.NoError(t, err)
	_, err = f.db.Exec(f.ctx, `INSERT INTO intakes(id,name,description,is_default,view_props,logo_props,project_id,workspace_id,created_at,updated_at) VALUES($1,'Intake','',false,'{}','{}',$2,$3,now(),now())`, intake, f.opts.ProjectID, f.opts.WorkspaceID)
	require.NoError(t, err)
	_, err = f.db.Exec(f.ctx, `INSERT INTO intake_issues(id,intake_id,issue_id,project_id,workspace_id,status,extra,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,2,'{}',now(),now())`, intake, item, f.opts.ProjectID, f.opts.WorkspaceID)
	require.NoError(t, err)
	_, err = f.db.Exec(f.ctx, `UPDATE issues SET archived_at=now() WHERE id=$1`, item)
	require.NoError(t, err)
	page, err := f.repo.ListWorkItems(f.ctx, f.opts.Scope, id, djangoapi.Query{})
	require.NoError(t, err)
	require.Len(t, page.Results, 1)
	_, err = f.repo.ManageWorkItems(f.ctx, f.opts, id, nil, []uuid.UUID{item})
	require.NoError(t, err)
	_, err = f.db.Exec(f.ctx, `UPDATE issues SET archived_at=NULL WHERE id=$1`, item)
	require.NoError(t, err)
	_, err = f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{item}, nil)
	require.ErrorIs(t, err, milestones.ErrInvalidInput, "intake ownership applies regardless of triage status")
	_, err = f.db.Exec(f.ctx, `UPDATE intake_issues SET deleted_at=now() WHERE issue_id=$1`, item)
	require.NoError(t, err)
	_, err = f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{item}, nil)
	require.NoError(t, err)
	f.counts(t, 1, 7, 0)
}
